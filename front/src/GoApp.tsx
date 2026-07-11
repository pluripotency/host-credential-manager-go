import { useState, useEffect } from 'react'

interface Toast {
  id: string;
  message: string;
  type: "success" | "error" | "info";
}

export function GoApp() {
  const [_toasts, setToasts] = useState<Toast[]>([]);
  const [loading, setLoading] = useState(true);
  const [message, setMessage] = useState('');

  // Show Toast Toast Notification helper
  const addToast = (message: string, type: Toast["type"] = "success") => {
    const id = Date.now().toString() + Math.random().toString().substring(2, 5);
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 3000);
  };
  const fetchHelloFromGo = async ()=>{
    setLoading(true);
    try {
      const res = await fetch("/api/hello")
      if (!res.ok) throw new Error("Failed to get hello from go")
      const data = await res.json();
      setMessage(data.message);
    } catch (err) {
      addToast("Failed to connect to backend server", "error");
    } finally {
      setLoading(false);
    }
  }
  useEffect(()=> {
    fetchHelloFromGo();
  }, []);
  return (
    <>
      <section id="center">
        <div>
          <h2>Hello from Go</h2>
          <p>
            Edit <code>go source</code>
          </p>
          <div className="flex-1 overflow-auto bg-white relative">
            {loading ? (
              <div className="absolute inset-0 flex flex-col items-center justify-center bg-white/80 z-10 gap-3">
                <span className="text-xs font-semibold text-slate-500">Loading hosts...</span>
              </div>
            ) : ( 
              <div className="absolute inset-0 flex flex-col items-center justify-center bg-white/80 z-10 gap-3">
                <span className="text-xs font-semibold text-slate-500">{message}</span>
              </div>
            )}
        </div>
        </div>
      </section>
    </>
    );
}

