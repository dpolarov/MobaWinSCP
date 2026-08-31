using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Management;
using System.Net;
using System.Net.NetworkInformation;
using System.Runtime.InteropServices;
using System.Text;

internal static class Program
{
    private const uint GW_OWNER = 4;
    private const int GWL_STYLE = -16;
    private const int GWL_EXSTYLE = -20;

    private delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);
    private delegate int EnumPropsExProc(IntPtr hwnd, string lpszString, IntPtr hData, UIntPtr dwData);

    [DllImport("user32.dll")]
    private static extern bool EnumWindows(EnumWindowsProc lpEnumFunc, IntPtr lParam);

    [DllImport("user32.dll")]
    private static extern bool EnumChildWindows(IntPtr hWndParent, EnumWindowsProc lpEnumFunc, IntPtr lParam);

    [DllImport("user32.dll")]
    private static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint lpdwProcessId);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    private static extern int GetWindowText(IntPtr hWnd, StringBuilder lpString, int nMaxCount);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    private static extern int GetClassName(IntPtr hWnd, StringBuilder lpClassName, int nMaxCount);

    [DllImport("user32.dll")]
    private static extern bool IsWindowVisible(IntPtr hWnd);

    [DllImport("user32.dll")]
    private static extern IntPtr GetWindow(IntPtr hWnd, uint uCmd);

    [DllImport("user32.dll", EntryPoint = "GetWindowLongPtrW")]
    private static extern IntPtr GetWindowLongPtr64(IntPtr hWnd, int nIndex);

    [DllImport("user32.dll", EntryPoint = "GetWindowLongW")]
    private static extern int GetWindowLong32(IntPtr hWnd, int nIndex);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    private static extern int EnumPropsEx(IntPtr hWnd, EnumPropsExProc lpEnumFunc, UIntPtr lParam);

    private static IntPtr GetWindowLongPtr(IntPtr hWnd, int nIndex)
        => IntPtr.Size == 8 ? GetWindowLongPtr64(hWnd, nIndex) : new IntPtr(GetWindowLong32(hWnd, nIndex));

    private sealed class ProcInfo
    {
        public int Pid;
        public int ParentPid;
        public string Name = "";
        public string CommandLine = "";
        public string ExecutablePath = "";
        public string MainWindowTitle = "";
    }

    private static int Main(string[] args)
    {
        Console.OutputEncoding = Encoding.UTF8;
        var mode = args.Length > 0 ? args[0].ToLowerInvariant() : "snapshot";
        var label = args.Length > 1 ? args[1] : DateTime.Now.ToString("yyyyMMdd-HHmmss");

        if (mode == "diff")
        {
            if (args.Length < 3)
            {
                Console.Error.WriteLine("Usage: MobaProbe.exe diff <snapshotA.txt> <snapshotB.txt>");
                return 2;
            }
            return Diff(args[1], args[2]);
        }

        if (mode != "snapshot")
        {
            Console.Error.WriteLine("Usage: MobaProbe.exe snapshot [label] | diff <A> <B>");
            return 2;
        }

        var outFile = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, $"mobaprobe-{Sanitize(label)}.txt");
        try
        {
            File.WriteAllText(outFile, BuildSnapshot(), new UTF8Encoding(false));
            Console.WriteLine(outFile);
            return 0;
        }
        catch (Exception ex)
        {
            Console.Error.WriteLine(ex);
            return 1;
        }
    }

    private static string BuildSnapshot()
    {
        var sb = new StringBuilder();
        sb.AppendLine("=== MobaProbe read-only snapshot ===");
        sb.AppendLine("Timestamp=" + DateTimeOffset.Now.ToString("O"));
        sb.AppendLine("Machine=" + Environment.MachineName);
        sb.AppendLine("User=" + Environment.UserName);
        sb.AppendLine();

        var all = ReadProcesses();
        var roots = all.Where(p => p.Name.Equals("MobaXterm.exe", StringComparison.OrdinalIgnoreCase) || p.Name.Equals("MobaXterm", StringComparison.OrdinalIgnoreCase)).ToList();
        if (roots.Count == 0)
        {
            sb.AppendLine("MobaXterm process not found.");
            return sb.ToString();
        }

        var interesting = new HashSet<int>();
        foreach (var root in roots)
            AddDescendants(root.Pid, all, interesting);

        sb.AppendLine("[PROCESSES]");
        foreach (var p in all.Where(p => interesting.Contains(p.Pid)).OrderBy(p => p.Pid))
        {
            sb.AppendLine($"PID={p.Pid} PPID={p.ParentPid} Name={p.Name}");
            sb.AppendLine($"  Path={p.ExecutablePath}");
            sb.AppendLine($"  Cmd={p.CommandLine}");
            sb.AppendLine($"  MainWindowTitle={p.MainWindowTitle}");
        }
        sb.AppendLine();

        sb.AppendLine("[WINDOWS]");
        foreach (var p in all.Where(p => interesting.Contains(p.Pid)).OrderBy(p => p.Pid))
            DumpWindowsForPid(sb, p.Pid);
        sb.AppendLine();

        sb.AppendLine("[TCP]");
        DumpTcp(sb, interesting);
        sb.AppendLine();

        sb.AppendLine("[TEMP_FILES]");
        DumpTempFiles(sb);
        sb.AppendLine();

        sb.AppendLine("[ENV_HINTS]");
        foreach (var key in new[] { "TEMP", "TMP", "APPDATA", "LOCALAPPDATA", "USERPROFILE" })
            sb.AppendLine($"{key}={Environment.GetEnvironmentVariable(key)}");

        return sb.ToString();
    }

    private static List<ProcInfo> ReadProcesses()
    {
        var result = new List<ProcInfo>();
        using (var searcher = new ManagementObjectSearcher("SELECT ProcessId,ParentProcessId,Name,CommandLine,ExecutablePath FROM Win32_Process"))
        using (var items = searcher.Get())
        {
            foreach (ManagementObject mo in items)
            {
                var p = new ProcInfo
                {
                    Pid = Convert.ToInt32((uint)mo["ProcessId"]),
                    ParentPid = Convert.ToInt32((uint)mo["ParentProcessId"]),
                    Name = Convert.ToString(mo["Name"]) ?? "",
                    CommandLine = Convert.ToString(mo["CommandLine"]) ?? "",
                    ExecutablePath = Convert.ToString(mo["ExecutablePath"]) ?? ""
                };
                try { p.MainWindowTitle = Process.GetProcessById(p.Pid).MainWindowTitle; } catch { }
                result.Add(p);
            }
        }
        return result;
    }

    private static void AddDescendants(int pid, List<ProcInfo> all, HashSet<int> set)
    {
        if (!set.Add(pid)) return;
        foreach (var child in all.Where(p => p.ParentPid == pid))
            AddDescendants(child.Pid, all, set);
    }

    private static void DumpWindowsForPid(StringBuilder sb, int pid)
    {
        EnumWindows((hwnd, _) =>
        {
            GetWindowThreadProcessId(hwnd, out var ownerPid);
            if (ownerPid != (uint)pid) return true;
            DumpWindow(sb, hwnd, "TOP");
            EnumChildWindows(hwnd, (child, __) =>
            {
                GetWindowThreadProcessId(child, out var childPid);
                if (childPid == (uint)pid)
                    DumpWindow(sb, child, "CHILD");
                return true;
            }, IntPtr.Zero);
            return true;
        }, IntPtr.Zero);
    }

    private static void DumpWindow(StringBuilder sb, IntPtr hwnd, string kind)
    {
        var title = new StringBuilder(2048);
        var cls = new StringBuilder(512);
        GetWindowText(hwnd, title, title.Capacity);
        GetClassName(hwnd, cls, cls.Capacity);
        var owner = GetWindow(hwnd, GW_OWNER);
        var style = unchecked((ulong)GetWindowLongPtr(hwnd, GWL_STYLE).ToInt64());
        var exstyle = unchecked((ulong)GetWindowLongPtr(hwnd, GWL_EXSTYLE).ToInt64());
        sb.AppendLine($"{kind} HWND=0x{hwnd.ToInt64():X} Visible={IsWindowVisible(hwnd)} Class='{cls}' Title='{title}' Owner=0x{owner.ToInt64():X} Style=0x{style:X} ExStyle=0x{exstyle:X}");
        try
        {
            EnumPropsEx(hwnd, (h, name, data, _) =>
            {
                sb.AppendLine($"  PROP Name='{name}' Data=0x{data.ToInt64():X}");
                return 1;
            }, UIntPtr.Zero);
        }
        catch (Exception ex)
        {
            sb.AppendLine("  PROP ERROR=" + ex.GetType().Name + ": " + ex.Message);
        }
    }

    private static void DumpTcp(StringBuilder sb, HashSet<int> interesting)
    {
        try
        {
            using (var searcher = new ManagementObjectSearcher("SELECT OwningProcess,LocalAddress,LocalPort,RemoteAddress,RemotePort,State FROM MSFT_NetTCPConnection", "root\\StandardCimv2"))
            using (var rows = searcher.Get())
            {
                foreach (ManagementObject mo in rows)
                {
                    var pid = Convert.ToInt32((uint)mo["OwningProcess"]);
                    if (!interesting.Contains(pid)) continue;
                    sb.AppendLine($"PID={pid} {mo["LocalAddress"]}:{mo["LocalPort"]} -> {mo["RemoteAddress"]}:{mo["RemotePort"]} State={mo["State"]}");
                }
            }
        }
        catch (Exception ex)
        {
            sb.AppendLine("TCP ERROR=" + ex.GetType().Name + ": " + ex.Message);
            try
            {
                foreach (var c in IPGlobalProperties.GetIPGlobalProperties().GetActiveTcpConnections())
                    sb.AppendLine($"UNKNOWN_PID {c.LocalEndPoint} -> {c.RemoteEndPoint} State={c.State}");
            }
            catch { }
        }
    }

    private static void DumpTempFiles(StringBuilder sb)
    {
        var roots = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var p in new[] { Environment.GetEnvironmentVariable("TEMP"), Environment.GetEnvironmentVariable("TMP"), Path.GetTempPath() })
            if (!string.IsNullOrWhiteSpace(p) && Directory.Exists(p)) roots.Add(Path.GetFullPath(p));

        foreach (var root in roots.OrderBy(x => x))
        {
            sb.AppendLine("ROOT=" + root);
            try
            {
                var entries = Directory.EnumerateFileSystemEntries(root, "*", SearchOption.TopDirectoryOnly)
                    .Select(path => new FileSystemInfoWrap(path))
                    .Where(x => x.Name.IndexOf("moba", StringComparison.OrdinalIgnoreCase) >= 0 || x.Name.IndexOf("ssh", StringComparison.OrdinalIgnoreCase) >= 0 || x.Name.IndexOf("sftp", StringComparison.OrdinalIgnoreCase) >= 0)
                    .OrderBy(x => x.Name)
                    .Take(500);
                foreach (var e in entries)
                    sb.AppendLine($"  {e.Kind} Name='{e.Name}' Size={e.Size} LastWrite={e.LastWrite:O}");
            }
            catch (Exception ex)
            {
                sb.AppendLine("  ERROR=" + ex.GetType().Name + ": " + ex.Message);
            }
        }
    }

    private sealed class FileSystemInfoWrap
    {
        public string Name { get; }
        public string Kind { get; }
        public long Size { get; }
        public DateTime LastWrite { get; }
        public FileSystemInfoWrap(string path)
        {
            var attr = File.GetAttributes(path);
            if ((attr & FileAttributes.Directory) != 0)
            {
                var di = new DirectoryInfo(path);
                Name = di.Name; Kind = "DIR"; Size = -1; LastWrite = di.LastWriteTime;
            }
            else
            {
                var fi = new FileInfo(path);
                Name = fi.Name; Kind = "FILE"; Size = fi.Length; LastWrite = fi.LastWriteTime;
            }
        }
    }

    private static int Diff(string a, string b)
    {
        if (!File.Exists(a) || !File.Exists(b))
        {
            Console.Error.WriteLine("Snapshot file not found.");
            return 2;
        }
        var A = new HashSet<string>(File.ReadAllLines(a).Where(IsStableLine));
        var B = new HashSet<string>(File.ReadAllLines(b).Where(IsStableLine));
        var outFile = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, $"mobaprobe-diff-{DateTime.Now:yyyyMMdd-HHmmss}.txt");
        using (var w = new StreamWriter(outFile, false, new UTF8Encoding(false)))
        {
            w.WriteLine("=== Only in A ===");
            foreach (var x in A.Except(B).OrderBy(x => x)) w.WriteLine(x);
            w.WriteLine();
            w.WriteLine("=== Only in B ===");
            foreach (var x in B.Except(A).OrderBy(x => x)) w.WriteLine(x);
        }
        Console.WriteLine(outFile);
        return 0;
    }

    private static bool IsStableLine(string s)
        => !s.StartsWith("Timestamp=", StringComparison.Ordinal) && !string.IsNullOrWhiteSpace(s);

    private static string Sanitize(string s)
    {
        foreach (var c in Path.GetInvalidFileNameChars()) s = s.Replace(c, '_');
        return s;
    }
}
