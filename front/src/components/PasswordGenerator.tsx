import { useState, useEffect } from "react";
import { Key, Copy, Check, RefreshCw, Shield } from "lucide-react";

interface PasswordGeneratorProps {
  onUsePassword?: (password: string) => void;
  className?: string;
}

export default function PasswordGenerator({ onUsePassword, className = "" }: PasswordGeneratorProps) {
  const [password, setPassword] = useState("");
  const [length, setLength] = useState(16);
  const [includeUppercase, setIncludeUppercase] = useState(true);
  const [includeLowercase, setIncludeLowercase] = useState(true);
  const [includeNumbers, setIncludeNumbers] = useState(true);
  const [includeSymbols, setIncludeSymbols] = useState(true);
  const [copied, setCopied] = useState(false);
  const [strength, setStrength] = useState({ score: 0, label: "None", color: "bg-gray-200 text-gray-700" });

  const generatePassword = async () => {
    if (!includeLowercase && !includeUppercase && !includeNumbers && !includeSymbols) {
      setPassword("Select at least one character type");
      setStrength({ score: 0, label: "None", color: "bg-gray-200 text-gray-700" });
      return;
    }

    try {
      const params = new URLSearchParams({
        length: length.toString(),
        lowercase: includeLowercase.toString(),
        uppercase: includeUppercase.toString(),
        numbers: includeNumbers.toString(),
        symbols: includeSymbols.toString(),
      });
      const res = await fetch(`/api/password/generate?${params}`);
      if (!res.ok) throw new Error("Failed to generate password");
      const data = await res.json();
      setPassword(data.password);
      setStrength(data.strength);
      setCopied(false);
    } catch (err) {
      console.error(err);
      setPassword("Failed to generate password from server");
      setStrength({ score: 0, label: "None", color: "bg-gray-200 text-gray-700" });
    }
  };

  // Generate initial password on load and update on settings changes
  useEffect(() => {
    generatePassword();
  }, [length, includeUppercase, includeLowercase, includeNumbers, includeSymbols]);

  const handleCopy = async () => {
    if (!password || password.startsWith("Select")) return;
    try {
      await navigator.clipboard.writeText(password);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy password:", err);
    }
  };

  return (
    <div className={`p-4 bg-white border border-slate-200 rounded-xl shadow-sm ${className}`}>
      <div className="flex items-center gap-2 mb-3">
        <div className="p-1.5 bg-blue-50 text-blue-600 rounded-lg">
          <Key className="w-4 h-4" />
        </div>
        <h3 className="text-sm font-semibold text-slate-800">Password Generator</h3>
      </div>

      {/* Password display bar */}
      <div className="flex items-center gap-1.5 mb-3 p-2 bg-slate-50 border border-slate-200 rounded-lg">
        <span className="flex-1 font-mono text-xs text-slate-800 break-all select-all font-medium">
          {password}
        </span>
        <div className="flex gap-1 shrink-0">
          <button
            onClick={generatePassword}
            title="Regenerate"
            className="p-1.5 hover:bg-slate-200 text-slate-500 rounded-md transition-colors"
          >
            <RefreshCw className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={handleCopy}
            disabled={password.startsWith("Select")}
            title="Copy Password"
            className={`p-1.5 rounded-md transition-colors ${
              copied
                ? "bg-emerald-50 text-emerald-600"
                : "hover:bg-slate-200 text-slate-500"
            }`}
          >
            {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          </button>
        </div>
      </div>

      {/* Strength indicator */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-1">
          <Shield className="w-3.5 h-3.5 text-slate-400" />
          <span className="text-xs text-slate-500">Security Strength:</span>
        </div>
        <span className={`px-2 py-0.5 rounded text-[10px] font-bold ${strength.color}`}>
          {strength.label}
        </span>
      </div>

      {/* Length Slider */}
      <div className="mb-4">
        <div className="flex justify-between text-xs text-slate-600 mb-1">
          <span>Password Length:</span>
          <span className="font-mono font-bold text-slate-800">{length} characters</span>
        </div>
        <input
          type="range"
          min="8"
          max="32"
          value={length}
          onChange={(e) => setLength(parseInt(e.target.value))}
          className="w-full accent-blue-600 cursor-pointer h-1.5 bg-slate-200 rounded-lg appearance-none"
        />
      </div>

      {/* Settings Grid */}
      <div className="grid grid-cols-2 gap-2 mb-4">
        <label className="flex items-center gap-2 text-xs text-slate-600 cursor-pointer hover:text-slate-800">
          <input
            type="checkbox"
            checked={includeLowercase}
            onChange={(e) => setIncludeLowercase(e.target.checked)}
            className="rounded border-slate-300 text-blue-600 focus:ring-blue-500 w-3.5 h-3.5"
          />
          <span>lowercase</span>
        </label>
        <label className="flex items-center gap-2 text-xs text-slate-600 cursor-pointer hover:text-slate-800">
          <input
            type="checkbox"
            checked={includeUppercase}
            onChange={(e) => setIncludeUppercase(e.target.checked)}
            className="rounded border-slate-300 text-blue-600 focus:ring-blue-500 w-3.5 h-3.5"
          />
          <span>UPPERCASE</span>
        </label>
        <label className="flex items-center gap-2 text-xs text-slate-600 cursor-pointer hover:text-slate-800">
          <input
            type="checkbox"
            checked={includeNumbers}
            onChange={(e) => setIncludeNumbers(e.target.checked)}
            className="rounded border-slate-300 text-blue-600 focus:ring-blue-500 w-3.5 h-3.5"
          />
          <span>Numbers (0-9)</span>
        </label>
        <label className="flex items-center gap-2 text-xs text-slate-600 cursor-pointer hover:text-slate-800">
          <input
            type="checkbox"
            checked={includeSymbols}
            onChange={(e) => setIncludeSymbols(e.target.checked)}
            className="rounded border-slate-300 text-blue-600 focus:ring-blue-500 w-3.5 h-3.5"
          />
          <span>Symbols (!@#)</span>
        </label>
      </div>

      {/* Direct Use CTA */}
      {onUsePassword && (
        <button
          onClick={() => onUsePassword(password)}
          disabled={password.startsWith("Select")}
          className="w-full text-center py-1.5 text-xs font-medium text-blue-600 hover:text-blue-700 hover:bg-blue-50 bg-transparent border border-dashed border-blue-300 rounded-lg transition-all"
        >
          Use This Password in Form
        </button>
      )}
    </div>
  );
}
