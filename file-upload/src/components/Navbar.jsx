import { useEffect, useState } from "react";

const Navbar = () => {
  const [status, setStatus] = useState("offline");

  useEffect(() => {
    const checkServerStatus = async () => {
      try {
        // Replace with your actual Go server endpoint
        const response = await fetch("http://localhost:8080/health");

        if (response.ok) {
          setStatus("ready");
        } else {
          setStatus("offline");
        }
      } catch (error) {
        console.error("Server is unreachable:", error);
        setStatus("offline");
      }
    };

    checkServerStatus();
  }, []);
  return (
    <>
      <nav className="w-full bg-white border-b border-gray-200 py-4 px-6 mb-8">
        <div className="max-w-xl mx-auto flex items-center justify-between">
          {/* Brand Title */}
          <span className="text-lg font-bold text-gray-800 tracking-tight flex items-center gap-2">
            <span className="text-xl" aria-hidden="true">
              📁
            </span>{" "}
            File Handling
          </span>

          {status === "ready" && (
            <span className="inline-flex items-center rounded-md bg-green-50 px-2 py-1 text-xs font-medium text-green-700 ring-1 ring-inset ring-green-600/20">
              <span className="mr-1.5 h-1.5 w-1.5 rounded-full bg-green-500" />
              Ready
            </span>
          )}

          {status === "offline" && (
            <span className="inline-flex items-center rounded-md bg-red-50 px-2 py-1 text-xs font-medium text-red-700 ring-1 ring-inset ring-red-600/20">
              <span className="mr-1.5 h-1.5 w-1.5 rounded-full bg-red-500 animate-pulse" />
              Offline
            </span>
          )}
        </div>
      </nav>
    </>
  );
};

export default Navbar;
