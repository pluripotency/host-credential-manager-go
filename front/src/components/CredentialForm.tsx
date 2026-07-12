import React, { useState, useEffect } from "react";
import type { HostList } from "../types";
import { X, Server, Key, Layers, Plus, Trash2 } from "lucide-react";
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

export default function CredentialForm({ credential, onSave, onCancel }: CredentialFormProps) {
  const isEdit = !!credential;
  
  const [hostname, setHostname] = useState("");
  const [ip, setIp] = useState("");
  const [platform, setPlatform] = useState("Linux");
  const [port, setPort] = useState("22");
  const [tags, setTags] = useState("");
  const [description, setDescription] = useState("");
  
  // State for multiple credentials
  const [userlist, setUserlist] = useState<UserCredential[]>([{ username: "", password: "" }]);

  // Initialize form with existing credential data if editing
  useEffect(() => {
    if (credential) {
      setHostname(credential.hostname);
      setIp(credential.ip);
      setPlatform(credential.platform);
      setPort(credential.port);
      setTags(credential.tags);
      setDescription(credential.description);
      if (credential.userlist && credential.userlist.length > 0) {
        // Deep copy of userlist array to avoid state side effects
        setUserlist(credential.userlist.map(u => ({ ...u })));
      } else {
        setUserlist([{ username: "", password: "" }]);
      }
    } else {
      resetForm();
    }
  }, [credential]);

  const resetForm = () => {
    setHostname("");
    setIp("");
    setPlatform("Linux");
    setPort("22");
    setTags("");
    setDescription("");
    setUserlist([{ username: "", password: "" }]);
  };

  // Handle auto-port mapping when platform changes
  const handlePlatformChange = (newPlatform: string) => {
    setPlatform(newPlatform);
    if (!isEdit && PLATFORM_PORTS[newPlatform]) {
      setPort(PLATFORM_PORTS[newPlatform]);
    }
  };

  const handleTagClick = (tag: string) => {
    const currentTags = tags ? tags.split(",").map((t) => t.trim()).filter(Boolean) : [];
    if (currentTags.includes(tag)) {
      // Remove tag
      setTags(currentTags.filter((t) => t !== tag).join(", "));
    } else {
      // Add tag
      setTags([...currentTags, tag].join(", "));
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!hostname.trim() || !platform.trim()) {
      alert("Please fill in all required fields (Hostname, Platform)");
      return;
    }

    // Filter out empty username/password rows
    const validUsers = userlist.filter(u => u.username.trim() && u.password.trim());
    if (validUsers.length === 0) {
      alert("Please add at least one Username/Password pair.");
      return;
    }

    onSave({
      ...(credential?.id ? { id: credential.id } : {}),
      hostname: hostname.trim(),
      ip: ip.trim(),
      platform,
      port: port.trim(),
      tags: tags.trim(),
      description: description.trim(),
      userlist: validUsers.map(u => ({
        username: u.username.trim(),
        password: u.password
      })),
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
              Port Number
            </label>
            <input
              type="text"
              value={port}
              onChange={(e) => setPort(e.target.value)}
              placeholder="e.g. 22"
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

        {/* Credentials Editor */}
        <div className="space-y-3">
          <div className="flex justify-between items-center">
            <label className="block text-xs font-semibold text-slate-600">
              User Credentials <span className="text-rose-500">*</span>
            </label>
            <button
              type="button"
              onClick={() => setUserlist([...userlist, { username: "", password: "" }])}
              className="text-[10px] bg-blue-50 border border-blue-200 text-blue-600 hover:bg-blue-100 px-2 py-1 rounded flex items-center gap-1 font-semibold transition-colors"
            >
              <Plus className="w-3 h-3" />
              Add User
            </button>
          </div>

          <div className="space-y-3">
            {userlist.map((user, idx) => (
              <div key={idx} className="p-3 bg-slate-50 border border-slate-200 rounded-xl space-y-2 relative">
                {userlist.length > 1 && (
                  <button
                    type="button"
                    onClick={() => setUserlist(userlist.filter((_, i) => i !== idx))}
                    className="absolute top-2 right-2 p-1 hover:bg-rose-100 text-slate-400 hover:text-rose-600 rounded transition-colors"
                    title="Remove User"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                )}
                
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
                    required
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
                        // Generate random clean password
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
                    required
                    className="w-full text-xs p-1.5 bg-white border border-slate-200 rounded-lg text-slate-800 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all font-mono"
                  />
                </div>
              </div>
            ))}
          </div>
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

          {/* Quick Tag Recommendations */}
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
