using System;
using System.Diagnostics;
using System.IO;
using System.Runtime.InteropServices;
using Accessibility;

internal static class Program
{
    private const uint ObjIdWindow = 0x00000000;

    [DllImport("oleacc.dll")]
    private static extern int AccessibleObjectFromWindow(
        IntPtr hwnd,
        uint dwId,
        ref Guid riid,
        [In, Out, MarshalAs(UnmanagedType.Interface)] ref object ppvObject);

    [DllImport("oleacc.dll")]
    private static extern int AccessibleChildren(
        IAccessible paccContainer,
        int iChildStart,
        int cChildren,
        [Out, MarshalAs(UnmanagedType.LPArray, SizeParamIndex = 2)] object[] rgvarChildren,
        out int pcObtained);

    private static StreamWriter log;

    private static string Safe(Func<object> getter)
    {
        try
        {
            object value = getter();
            return value == null ? "" : value.ToString();
        }
        catch (Exception ex)
        {
            return "<" + ex.GetType().Name + ": " + ex.Message + ">";
        }
    }

    private static void Dump(IAccessible acc, int depth)
    {
        if (acc == null || depth > 25)
            return;

        string pad = new string(' ', depth * 2);
        int count = 0;
        try { count = acc.accChildCount; }
        catch (Exception ex)
        {
            log.WriteLine(pad + "accChildCount ERROR: " + ex.Message);
        }

        log.WriteLine(
            pad + "[OBJECT] " +
            "Name='" + Safe(delegate { return acc.get_accName(0); }) + "' " +
            "Value='" + Safe(delegate { return acc.get_accValue(0); }) + "' " +
            "Description='" + Safe(delegate { return acc.get_accDescription(0); }) + "' " +
            "Role='" + Safe(delegate { return acc.get_accRole(0); }) + "' " +
            "State='" + Safe(delegate { return acc.get_accState(0); }) + "' " +
            "Children=" + count);

        if (count <= 0)
            return;

        object[] children = new object[count];
        int obtained = 0;
        int hr;

        try
        {
            hr = AccessibleChildren(acc, 0, count, children, out obtained);
        }
        catch (Exception ex)
        {
            log.WriteLine(pad + "AccessibleChildren ERROR: " + ex);
            return;
        }

        log.WriteLine(pad + "AccessibleChildren HR=0x" + hr.ToString("X8") + " Obtained=" + obtained);

        for (int i = 0; i < obtained; i++)
        {
            object child = children[i];
            IAccessible childAcc = child as IAccessible;

            if (childAcc != null)
            {
                log.WriteLine(pad + "  CHILD[" + i + "] IAccessible type=" + child.GetType().FullName);
                Dump(childAcc, depth + 1);
                continue;
            }

            object id = child;
            log.WriteLine(
                pad + "  CHILD[" + i + "] ID=" + (id == null ? "<null>" : id.ToString()) + " " +
                "Type='" + (id == null ? "" : id.GetType().FullName) + "' " +
                "Name='" + Safe(delegate { return acc.get_accName(id); }) + "' " +
                "Value='" + Safe(delegate { return acc.get_accValue(id); }) + "' " +
                "Description='" + Safe(delegate { return acc.get_accDescription(id); }) + "' " +
                "Role='" + Safe(delegate { return acc.get_accRole(id); }) + "' " +
                "State='" + Safe(delegate { return acc.get_accState(id); }) + "'");
        }
    }

    private static int Main()
    {
        string output = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "mobaxterm-msaa.txt");

        using (log = new StreamWriter(output, false))
        {
            Process[] processes = Process.GetProcessesByName("MobaXterm");
            if (processes.Length == 0)
            {
                log.WriteLine("MobaXterm not found");
                Console.Error.WriteLine("MobaXterm not found");
                return 1;
            }

            Process p = null;
            foreach (Process candidate in processes)
            {
                if (candidate.MainWindowHandle != IntPtr.Zero)
                {
                    p = candidate;
                    break;
                }
            }
            if (p == null)
                p = processes[0];

            log.WriteLine("PID=" + p.Id);
            log.WriteLine("HWND=0x" + p.MainWindowHandle.ToInt64().ToString("X"));
            log.WriteLine("Title=" + p.MainWindowTitle);

            Guid iid = new Guid("618736E0-3C3D-11CF-810C-00AA00389B71");
            object obj = null;
            int hr = AccessibleObjectFromWindow(p.MainWindowHandle, ObjIdWindow, ref iid, ref obj);

            log.WriteLine("AccessibleObjectFromWindow HR=0x" + hr.ToString("X8"));
            log.WriteLine("Object null=" + (obj == null));
            log.WriteLine("Object type=" + (obj == null ? "" : obj.GetType().FullName));

            IAccessible acc = obj as IAccessible;
            if (acc == null)
            {
                log.WriteLine("Returned object does not implement IAccessible");
                return 2;
            }

            log.WriteLine("IAccessible cast OK");
            Dump(acc, 0);
        }

        Console.WriteLine(output);
        return 0;
    }
}
