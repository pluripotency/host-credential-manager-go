import React, { useState, useEffect } from "react";
import type { HostList, AccessItem } from "../types";
import { X, Server, Key, Layers, Plus, Trash2, Globe, ExternalLink } from "lucide-react";
import { platformOptions, PLATFORM_PORTS } from "./PlatformDefs";

interface UserCredential {
  username: string;
  password: string;
}

interface CredentialFormProps {
  credential?: HostList | null;
  onSave: (data: any) => void;
  onCancel: () => void;
}

const POPULAR_TAGS = ["production", "staging", "internal", "web", "database", "network", "devops", "infrastructure"];

const PROTOCOL_OPTIONS = [
  { label: "HTTP", value: "http", defaultPort: "80" },
  { label: "HTTPS", value: "https", defaultPort: "443" },
  { label: "SSH", value: "ssh", defaultPort: "22" },
  { label: "Telnet", value: "telnet", defaultPort: "23" },
  { label: "RDP", value: "rdp", defaultPort: "3389" },
  { label: "MySQL", value: "mysql", defaultPort: "3306" },
  { label: "PostgreSQL", value: "postgres", defaultPort: "5432" },
  { label: "Redis", value: "redis", defaultPort: "6379" },
  { label: "MongoDB", value: "mongodb", defaultPort: "27017" },
  { label: "LDAP", value: "ldap", defaultPort: "389" },
  { label: "Oracle", value: "oracle", defaultPort: "1521" },
];

export default function CredentialForm({ credential, onSave, onCancel }: CredentialFormProps) {
  const isEdit = !!credential;
  
  const [hostname, setHostname] = useState("");
  const [ip, setIp] = useState("");
  const [platform, setPlatform] = useState("Linux");
  const [os, setOs] = useState("");
  const [tags, setTags] = useState("");
  const [description, setDescription] = useState("");
  
  // State for AccessList
  const [accesslist, setAccesslist] = useState<AccessItem[]>([{ protocol: "ssh", port: "22", path: "" }]);

  // State for user credentials (optional, can be empty)
  const [userlist, setUserlist] = useState<UserCredential[]>([]);

  // Initialize form with existing credential data if editing
  useEffect(() => {
    if (credential) {
      setHostname(credential.hostname);
      setIp(credential.ip);
      setPlatform(credential.platform);
      setOs(credential.os || "");
      setTags(credential.tags);
      setDescription(credential.description);
      
      if (credential.accesslist && credential.accesslist.length > 0) {
        setAccesslist(credential.accesslist.map(a => ({ ...a })));
      } else {
        const defaultProto = PLATFORM_PORTS[credential.platform] === "443" ? "https" :
                             PLATFORM_PORTS[credential.platform] === "80" ? "http" : "ssh";
        setAccesslist([{ protocol: defaultProto, port: credential.port || "22", path: "" }]);
      }

      if (credential.userlist && credential.userlist.length > 0) {
        setUserlist(credential.userlist.map(u => ({ ...u })));
      } else {
        setUserlist([]);
      }
    } else {
      resetForm();
    }
  }, [credential]);

  const resetForm = () => {
    setHostname("");
    setIp("");
    setPlatform("Linux");
    setOs("");
    setTags("");
    setDescription("");
    setAccesslist([{ protocol: "ssh", port: "22", path: "" }]);
    setUserlist([]);
  };

  const handlePlatformChange = (newPlatform: string) => {
    setPlatform(newPlatform);
    if (!isEdit && PLATFORM_PORTS[newPlatform]) {
      const defaultPort = PLATFORM_PORTS[newPlatform];
      if (accesslist.length === 1 && (!accesslist[0].port || accesslist[0].port === "22")) {
        setAccesslist([{ ...accesslist[0], port: defaultPort }]);
      }
    }
  };

  const handleTagClick = (tag: string) => {
    const currentTags = tags ? tags.split(",").map((t) => t.trim()).filter(Boolean) : [];
    if (currentTags.includes(tag)) {
      setTags(currentTags.filter((t) => t !== tag).join(", "));
    } else {
      setTags([...currentTags, tag].join(", "));
    }
  };

  const computeFullUrl = (item: AccessItem, currentIp: string, currentHost: string) => {
    const target = currentIp.trim() || currentHost.trim() || "localhost";
    const proto = (item.protocol || "http").toLowerCase().trim();
    const portStr = item.port.trim() ? `:${item.port.trim()}` : "";
    let rawPath = (item.path || "").trim();
    if (rawPath && !rawPath.startsWith("/")) {
      rawPath = "/" + rawPath;
    }
    return `${proto}://${target}${portStr}${rawPath}`;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!hostname.trim() || !platform.trim()) {
      alert("Please fill in all required fields (Hostname, Platform)");
      return;
    }

    const validUsers = userlist
      .filter(u => u.username.trim() || u.password.trim())
      .map(u => ({
        username: u.username.trim(),
        password: u.password
      }));

    const validAccess = accesslist.filter(a => a.protocol.trim() && a.port.trim());
    if (validAccess.length === 0) {
      alert("Please add at least one Access protocol/port entry.");
      return;
    }

    onSave({
      ...(credential?.id ? { id: credential.id } : {}),
      hostname: hostname.trim(),
      ip: ip.trim(),
      platform,
      os: os.trim(),
      port: validAccess[0].port.trim(),
      tags: tags.trim(),
      description: description.trim(),
      accesslist: validAccess.map(a => ({
        protocol: a.protocol.trim(),
        port: a.port.trim(),
        path: (a.path || "").trim(),
      })),
      userlist: validUsers,
    });
  };

  const currentTagsArray = tags ? tags.split(",").map((t) => t.trim()).filter(Boolean) : [];

  return (
    <div className="flex flex-col h-full bg-white max-w-lg w-full ml-auto shadow-2xl border-l border-slate-200">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-slate-100 bg-slate-50">
        <div className="flex items-center gap-2">
          <div className={`p-2 rounded-lg ${isEdit ? "bg-amber-50 text-amber-600" : "bg-blue-50 text-blue-600"}`}>
            {isEdit ? <Layers className="w-5 h-5" /> : <Server className="w-5 h-5" />}
          </div>
          <div>
            <h2 className="text-sm font-bold text-slate-800">
              {isEdit ? "Edit Host Details" : "Register Host"}
            </h2>
            <p className="text-[11px] text-slate-500">
              {isEdit ? "Update details for the selected server" : "Register a new server and credentials"}
            </p>
          </div>
        </div>
        <button
          onClick={onCancel}
          className="p-1.5 hover:bg-slate-200 text-slate-400 hover:text-slate-600 rounded-lg transition-colors"
        >
          <X className="w-4 h-4" />
        </button>
      </div>

      {/* Form Content */}
      <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Basic Identification Grid */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-semibold text-slate-600 mb-1">
              Platform <span className="text-rose-500">*</span>
            </label>
            <select
              value={platform}
              onChange={(e) => handlePlatformChange(e.target.value)}
              className="w-full text-xs p-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
            >
              {platformOptions.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-600 mb-1">
              OS <span className="text-slate-400 font-normal">(Optional)</span>
            </label>
            <input
              type="text"
              value={os}
              onChange={(e) => setOs(e.target.value)}
              placeholder="e.g. Ubuntu 24.04, Windows 11"
              className="w-full text-xs p-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono"
            />
          </div>
        </div>

        {/* Hostname & IP */}
        <div className="grid grid-cols-2 gap-3">
          <div className="col-span-2 sm:col-span-1">
            <label className="block text-xs font-semibold text-slate-600 mb-1">
              Hostname / Domain <span className="text-rose-500">*</span>
            </label>
            <input
              type="text"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
              placeholder="e.g. srv-prod-web01"
              required
              className="w-full text-xs p-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono"
            />
          </div>

          <div className="col-span-2 sm:col-span-1">
            <label className="block text-xs font-semibold text-slate-600 mb-1">
              IP Address
            </label>
            <input
              type="text"
              value={ip}
              onChange={(e) => setIp(e.target.value)}
              placeholder="e.g. 192.168.1.50"
              className="w-full text-xs p-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono"
            />
          </div>
        </div>

        <hr className="border-slate-100" />

        {/* AccessList Editor */}
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <div>
              <label className="block text-xs font-semibold text-slate-600">
                Access List (Protocols & Ports) <span className="text-rose-500">*</span>
              </label>
              <p className="text-[10px] text-slate-400">e.g. http(3000), https(8080), ssh(10022)</p>
            </div>
            <button
              type="button"
              onClick={() => setAccesslist([...accesslist, { protocol: "http", port: "8080", path: "" }])}
              className="text-[10px] bg-blue-50 border border-blue-200 text-blue-600 hover:bg-blue-100 px-2 py-1 rounded flex items-center gap-1 font-semibold transition-colors"
            >
              <Plus className="w-3 h-3" />
              Add Access
            </button>
          </div>

          <div className="space-y-3">
            {accesslist.map((item, idx) => {
              const protoLower = (item.protocol || "").toLowerCase().trim();
              const isWeb = protoLower === "http" || protoLower === "https";
              const fullUrl = computeFullUrl(item, ip, hostname);

              return (
                <div key={idx} className="p-3 bg-slate-50 border border-slate-200 rounded-xl space-y-2 relative">
                  {accesslist.length > 1 && (
                    <button
                      type="button"
                      onClick={() => setAccesslist(accesslist.filter((_, i) => i !== idx))}
                      className="absolute top-2 right-2 p-1 hover:bg-rose-100 text-slate-400 hover:text-rose-600 rounded transition-colors"
                      title="Remove Access Entry"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  )}

                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="block text-[10px] font-bold text-slate-400 mb-0.5 uppercase tracking-tight">
                        Protocol
                      </label>
                      <input
                        type="text"
                        list={`proto-options-${idx}`}
                        value={item.protocol}
                        onChange={(e) => {
                          const val = e.target.value;
                          const matched = PROTOCOL_OPTIONS.find((p) => p.value.toLowerCase() === val.toLowerCase());
                          const updated = [...accesslist];
                          updated[idx].protocol = val;
                          if (matched && (!updated[idx].port || updated[idx].port === "22" || updated[idx].port === "80")) {
                            updated[idx].port = matched.defaultPort;
                          }
                          setAccesslist(updated);
                        }}
                        placeholder="e.g. http, https, ssh"
                        className="w-full text-xs p-1.5 bg-white border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
                      />
                      <datalist id={`proto-options-${idx}`}>
                        {PROTOCOL_OPTIONS.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {opt.label}
                          </option>
                        ))}
                      </datalist>
                    </div>

                    <div>
                      <label className="block text-[10px] font-bold text-slate-400 mb-0.5 uppercase tracking-tight">
                        Port
                      </label>
                      <input
                        type="text"
                        value={item.port}
                        onChange={(e) => {
                          const updated = [...accesslist];
                          updated[idx].port = e.target.value;
                          setAccesslist(updated);
                        }}
                        placeholder="e.g. 3000"
                        className="w-full text-xs p-1.5 bg-white border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
                      />
                    </div>
                  </div>

                  {isWeb && (
                    <div className="pt-1 space-y-1.5">
                      <div>
                        <label className="block text-[10px] font-bold text-slate-400 mb-0.5 uppercase tracking-tight">
                          URL Path (Optional for HTTP/HTTPS)
                        </label>
                        <input
                          type="text"
                          value={item.path || ""}
                          onChange={(e) => {
                            const updated = [...accesslist];
                            updated[idx].path = e.target.value;
                            setAccesslist(updated);
                          }}
                          placeholder="e.g. /dashboard or /admin"
                          className="w-full text-xs p-1.5 bg-white border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono"
                        />
                      </div>

                      {/* Full URL Live Display */}
                      <div className="bg-blue-50/90 border border-blue-200 rounded-lg p-2 flex items-center justify-between gap-2 text-xs">
                        <div className="flex items-center gap-1.5 min-w-0">
                          <Globe className="w-3.5 h-3.5 text-blue-600 shrink-0" />
                          <span className="text-[10px] font-bold text-blue-900 shrink-0">Full URL:</span>
                          <span className="font-mono text-xs text-blue-700 font-semibold truncate" title={fullUrl}>
                            {fullUrl}
                          </span>
                        </div>
                        <a
                          href={fullUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="p-1 text-blue-600 hover:text-blue-800 hover:bg-blue-100 rounded transition-colors shrink-0"
                          title="Open URL in new tab"
                        >
                          <ExternalLink className="w-3.5 h-3.5" />
                        </a>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        <hr className="border-slate-100" />

        {/* Credentials Editor (Optional) */}
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <div>
              <label className="block text-xs font-semibold text-slate-600">
                User Credentials <span className="text-slate-400 font-normal">(Optional)</span>
              </label>
              <p className="text-[10px] text-slate-400">Username / Password pairs</p>
            </div>
            <button
              type="button"
              onClick={() => setUserlist([...userlist, { username: "", password: "" }])}
              className="text-[10px] bg-blue-50 border border-blue-200 text-blue-600 hover:bg-blue-100 px-2 py-1 rounded flex items-center gap-1 font-semibold transition-colors"
            >
              <Plus className="w-3 h-3" />
              Add User
            </button>
          </div>

          {userlist.length === 0 ? (
            <div className="p-3 bg-slate-50 border border-dashed border-slate-200 rounded-xl text-center">
              <p className="text-[11px] text-slate-400">No user credentials added for this host.</p>
            </div>
          ) : (
            <div className="space-y-3">
              {userlist.map((user, idx) => (
                <div key={idx} className="p-3 bg-slate-50 border border-slate-200 rounded-xl space-y-2 relative">
                  <button
                    type="button"
                    onClick={() => setUserlist(userlist.filter((_, i) => i !== idx))}
                    className="absolute top-2 right-2 p-1 hover:bg-rose-100 text-slate-400 hover:text-rose-600 rounded transition-colors"
                    title="Remove User"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                  
                  <div>
                    <label className="block text-[10px] font-bold text-slate-400 mb-0.5 uppercase tracking-tight">
                      Username
                    </label>
                    <input
                      type="text"
                      value={user.username}
                      onChange={(e) => {
                        const updated = [...userlist];
                        updated[idx].username = e.target.value;
                        setUserlist(updated);
                      }}
                      placeholder="e.g. root or administrator"
                      className="w-full text-xs p-1.5 bg-white border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all font-mono"
                    />
                  </div>

                  <div>
                    <div className="flex justify-between items-center mb-0.5">
                      <label className="block text-[10px] font-bold text-slate-400 uppercase tracking-tight">
                        Password
                      </label>
                      <button
                        type="button"
                        onClick={() => {
                          const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*";
                          let gen = "";
                          for (let i = 0; i < 16; i++) {
                            gen += chars.charAt(Math.floor(Math.random() * chars.length));
                          }
                          const updated = [...userlist];
                          updated[idx].password = gen;
                          setUserlist(updated);
                        }}
                        className="text-[9px] text-blue-600 hover:underline flex items-center gap-0.5"
                      >
                        <Key className="w-2.5 h-2.5" />
                        Generate
                      </button>
                    </div>
                    <input
                      type="text"
                      value={user.password}
                      onChange={(e) => {
                        const updated = [...userlist];
                        updated[idx].password = e.target.value;
                        setUserlist(updated);
                      }}
                      placeholder="Enter password"
                      className="w-full text-xs p-1.5 bg-white border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all font-mono"
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <hr className="border-slate-100" />

        {/* Tags */}
        <div>
          <label className="block text-xs font-semibold text-slate-600 mb-1">
            Tags (comma-separated)
          </label>
          <input
            type="text"
            value={tags}
            onChange={(e) => setTags(e.target.value)}
            placeholder="e.g. production, database, frontend"
            className="w-full text-xs p-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
          />

          <div className="mt-2">
            <span className="text-[10px] text-slate-400 block mb-1">Popular Tags:</span>
            <div className="flex flex-wrap gap-1">
              {POPULAR_TAGS.map((tag) => {
                const isActive = currentTagsArray.includes(tag);
                return (
                  <button
                    key={tag}
                    type="button"
                    onClick={() => handleTagClick(tag)}
                    className={`text-[10px] px-2 py-0.5 rounded-full border transition-all ${
                      isActive
                        ? "bg-blue-50 border-blue-200 text-blue-600 font-medium"
                        : "bg-white border-slate-200 text-slate-500 hover:bg-slate-50"
                    }`}
                  >
                    {tag}
                  </button>
                );
              })}
            </div>
          </div>
        </div>

        {/* Description / Note */}
        <div>
          <label className="block text-xs font-semibold text-slate-600 mb-1">
            Description / Notes
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Add location, usage details, or security notices here..."
            rows={3}
            className="w-full text-xs p-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all resize-none"
          />
        </div>
      </form>

      {/* Footer Controls */}
      <div className="p-4 bg-slate-50 border-t border-slate-100 flex justify-end gap-2 shrink-0">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-1.5 text-xs text-slate-600 hover:bg-slate-200 rounded-lg transition-colors font-medium"
        >
          Cancel
        </button>
        <button
          onClick={handleSubmit}
          className={`px-5 py-1.5 text-xs text-white font-medium rounded-lg shadow-sm transition-all ${
            isEdit
              ? "bg-amber-600 hover:bg-amber-700 hover:shadow"
              : "bg-blue-600 hover:bg-blue-700 hover:shadow"
          }`}
        >
          {isEdit ? "Update Host" : "Add Host"}
        </button>
      </div>
    </div>
  );
}
