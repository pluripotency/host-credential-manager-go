import React, { useState, useEffect, useRef } from "react";
import { 
  Server, Key, Search, Plus, Edit2, Trash2, Copy, Check, Eye, EyeOff, 
  Download, Upload, RefreshCw, X, 
  Lock, ArrowUpDown, 
  AlertCircle, Info
} from "lucide-react";
import './index.css'
import type { HostList, TableDensity } from "./types";
import CredentialForm from "./components/CredentialForm";
import { platformIcons, platformBadgeColors, CATEGORIES } from "./components/PlatformDefs";

interface Toast {
  id: string;
  message: string;
  type: "success" | "error" | "info";
}



export default function ManagerApp() {
  const [hostList, setHostList] = useState<HostList[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("All");
  const [selectedTag, setSelectedTag] = useState<string | null>(null);
  
  // Table state
  const [sortColumn, setSortColumn] = useState<keyof HostList>("hostname");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [density, setDensity] = useState<TableDensity>("dense");
  
  // Mask & reveal states
  const [revealedPasswords, setRevealedPasswords] = useState<Record<string, boolean>>({});

  // Form Drawer & Modal states
  const [formMode, setFormMode] = useState<"closed" | "add" | "edit">("closed");
  const [editingHost, setEditingHost] = useState<HostList | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [isImportOpen, setIsImportOpen] = useState(false);
  const [importMergeOption, setImportMergeOption] = useState<"merge" | "overwrite">("merge");
  
  // UI Utilities
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [copiedStates, setCopiedStates] = useState<Record<string, boolean>>({}); // e.g. "id-fieldname" -> true

  const fileInputRef = useRef<HTMLInputElement>(null);

  // Fetch all hosts
  const fetchHostList = async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/hostlist");
      if (!res.ok) throw new Error("Failed to load host database");
      const data = await res.json();
      setHostList(data);
    } catch (err) {
      addToast("Failed to connect to backend server", "error");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHostList();
  }, []);

  // Show Toast Toast Notification helper
  const addToast = (message: string, type: Toast["type"] = "success") => {
    const id = Date.now().toString() + Math.random().toString().substring(2, 5);
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 3000);
  };

  // Safe Clipboard Copy Handler
  const handleCopyToClipboard = async (text: string, key: string, label: string) => {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopiedStates((prev) => ({ ...prev, [key]: true }));
      addToast(`${label} copied to clipboard!`, "success");
      
      setTimeout(() => {
        setCopiedStates((prev) => ({ ...prev, [key]: false }));
      }, 2000);
    } catch (err) {
      addToast(`Failed to copy ${label.toLowerCase()}`, "error");
    }
  };

  // Delete Host
  const handleDelete = async (id: string) => {
    try {
      const res = await fetch(`/api/hostlist/${id}`, { method: "DELETE" });
      if (!res.ok) throw new Error("Delete failed");
      
      setHostList((prev) => prev.filter((h) => h.id !== id));
      addToast("Host successfully deleted", "success");
      setDeletingId(null);
    } catch (err) {
      addToast("Failed to delete host", "error");
    }
  };

  // Save Host (Create/Update)
  const handleSave = async (data: any) => {
    const isEdit = !!data.id;
    const url = isEdit ? `/api/hostlist/${data.id}` : "/api/hostlist";
    const method = isEdit ? "PUT" : "POST";

    try {
      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });

      if (!res.ok) {
        const errData = await res.json();
        throw new Error(errData.error || "Save failed");
      }

      const savedHost = await res.json();
      
      if (isEdit) {
        setHostList((prev) => prev.map((h) => (h.id === savedHost.id ? savedHost : h)));
        addToast("Host updated successfully", "success");
      } else {
        setHostList((prev) => [savedHost, ...prev]);
        addToast("New host added successfully", "success");
      }

      setFormMode("closed");
      setEditingHost(null);
    } catch (err: any) {
      addToast(err.message || "Failed to save host", "error");
    }
  };

  // Trigger file dialog
  const triggerImport = () => {
    fileInputRef.current?.click();
  };

  // Handle CSV upload
  const handleCSVUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = async (evt) => {
      const text = evt.target?.result;
      if (typeof text !== "string") return;

      // Parse with PapaParse client side first for client validations
      import("papaparse").then((Papa) => {
        const parsed = Papa.parse(text, { header: true, skipEmptyLines: true });
        if (parsed.errors.length > 0) {
          addToast("Error parsing CSV file locally", "error");
          return;
        }

        const data = parsed.data as any[];
        // Verify we have at least hostname and platform
        const hasRequired = data.every(row => row.hostname && row.platform);
        if (!hasRequired && data.length > 0) {
          if (!confirm("Some rows appear to be missing critical fields (hostname, platform). Do you want to try importing anyway?")) {
            return;
          }
        }

        // Post to backend import endpoint
        fetch("/api/hostlist/import", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            data,
            merge: importMergeOption === "merge"
          })
        })
        .then(res => {
          if (!res.ok) throw new Error("Import failed on server");
          return res.json();
        })
        .then(resData => {
          addToast(`Successfully imported ${resData.count} hosts!`, "success");
          setIsImportOpen(false);
          fetchHostList();
        })
        .catch(() => {
          addToast("Failed to upload CSV to server", "error");
        });
      });
    };
    reader.readAsText(file);
    e.target.value = ""; // clear file input
  };

  // HostList filtered ONLY by selectedCategory (used to gather available tags for this category)
  const categoryFilteredHostList = hostList.filter((h) => {
    if (selectedCategory !== "All") {
      const category = CATEGORIES.find((cat) => cat.name === selectedCategory);
      if (category) {
        return category.match(h);
      }
      return h.platform === selectedCategory;
    }
    return true;
  });

  // Extract unique tags from category-filtered hosts
  const availableTags = Array.from(
    new Set(
      categoryFilteredHostList
        .flatMap((h) => (h.tags ? h.tags.split(",").map((t) => t.trim()) : []))
        .filter((t) => t.length > 0)
    )
  ).sort();

  // Dynamic filter lists
  const filteredHostList = hostList.filter((h) => {
    // 1. Filter by category
    if (selectedCategory !== "All") {
      const category = CATEGORIES.find((cat) => cat.name === selectedCategory);
      if (category) {
        if (!category.match(h)) return false;
      } else {
        // Direct platform filter
        if (h.platform !== selectedCategory) return false;
      }
    }

    // 1b. Filter by recursively selected tag
    if (selectedTag) {
      if (!h.tags) return false;
      const tagsList = h.tags.split(",").map((t) => t.trim().toLowerCase());
      if (!tagsList.includes(selectedTag.toLowerCase())) return false;
    }

    // 2. Filter by search query (incremental search on hostname, IP, username, tags, platform, description)
    if (searchQuery.trim() !== "") {
      const q = searchQuery.toLowerCase();
      const matchHostname = (h.hostname || "").toLowerCase().includes(q);
      const matchIP = (h.ip || "").toLowerCase().includes(q);
      const matchPlatform = (h.platform || "").toLowerCase().includes(q);
      const matchTags = (h.tags || "").toLowerCase().includes(q);
      const matchDesc = (h.description || "").toLowerCase().includes(q);
      
      // Match if any user in the userlist matches the username
      const matchUsername = h.userlist
        ? h.userlist.some((u: any) => (u.username || "").toLowerCase().includes(q))
        : false;
      
      return matchHostname || matchIP || matchPlatform || matchTags || matchDesc || matchUsername;
    }

    return true;
  });

  // Dynamic sorting lists
  const sortedHostList = [...filteredHostList].sort((a, b) => {
    let aVal = a[sortColumn] || "";
    let bVal = b[sortColumn] || "";

    // handle case insensitivity
    if (typeof aVal === "string") aVal = aVal.toLowerCase();
    if (typeof bVal === "string") bVal = bVal.toLowerCase();

    if (aVal < bVal) return sortDirection === "asc" ? -1 : 1;
    if (aVal > bVal) return sortDirection === "asc" ? 1 : -1;
    return 0;
  });

  const handleSort = (column: keyof HostList) => {
    if (sortColumn === column) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
    } else {
      setSortColumn(column);
      setSortDirection("asc");
    }
  };

  // Helper to count hosts matching a category's rule dynamically
  const getCategoryCount = (name: string) => {
    const cat = CATEGORIES.find((c) => c.name === name);
    return cat ? hostList.filter(cat.match).length : 0;
  };

  return (
    <div className="min-h-screen flex flex-col bg-slate-50 text-slate-800 font-sans antialiased">
      {/* Toast Overlay */}
      <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm w-full">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`p-3 rounded-xl shadow-lg border text-xs font-semibold flex items-center justify-between gap-3 animate-slide-in ${
              toast.type === "error"
                ? "bg-rose-50 border-rose-200 text-rose-800"
                : toast.type === "info"
                ? "bg-indigo-50 border-indigo-200 text-indigo-800"
                : "bg-emerald-50 border-emerald-200 text-emerald-800"
            }`}
          >
            <div className="flex items-center gap-2">
              <div className={`p-1 rounded-full ${
                toast.type === "error"
                  ? "bg-rose-200 text-rose-800"
                  : toast.type === "info"
                  ? "bg-indigo-200 text-indigo-800"
                  : "bg-emerald-200 text-emerald-800"
              }`}>
                {toast.type === "error" ? (
                  <AlertCircle className="w-3.5 h-3.5" />
                ) : toast.type === "info" ? (
                  <Info className="w-3.5 h-3.5" />
                ) : (
                  <Check className="w-3.5 h-3.5" />
                )}
              </div>
              <span className="leading-tight">{toast.message}</span>
            </div>
            <button
              onClick={() => setToasts((prev) => prev.filter((t) => t.id !== toast.id))}
              className="text-slate-400 hover:text-slate-600 transition-colors shrink-0"
            >
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
      </div>

      {/* Main Header */}
      <header className="bg-slate-900 text-white shrink-0 shadow-md">
        <div className="px-4 py-3.5 flex flex-col sm:flex-row items-center justify-between gap-3 max-w-full">
          {/* Logo & Info */}
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-600 rounded-xl shadow-md shadow-blue-500/20">
              <Server className="w-5 h-5 text-white" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="font-extrabold text-sm tracking-tight leading-none">Host Database</h1>
                <span className="text-[10px] bg-slate-800 border border-slate-700 text-slate-300 font-mono px-1.5 py-0.2 rounded-full font-bold">
                  v2.0
                </span>
              </div>
              <p className="text-[10px] text-slate-400 mt-1">
                Unified Server Inventory & User Credential Management
              </p>
            </div>
          </div>

          {/* Quick Actions */}
          <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
            {/* Reload Button */}
            <button
              onClick={fetchHostList}
              disabled={loading}
              title="Reload from CSV Database"
              className="p-1.5 bg-white hover:bg-slate-100 border border-slate-200 rounded-lg text-slate-500 hover:text-slate-800 transition-colors flex items-center gap-1.5 text-xs font-semibold"
            >
              <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} />
              <span>Reload</span>
            </button>

            {/* Import Button */}
            <button
              onClick={() => setIsImportOpen(true)}
              className="p-1.5 bg-white hover:bg-slate-100 border border-slate-200 rounded-lg text-slate-600 hover:text-slate-800 transition-colors flex items-center gap-1.5 text-xs font-semibold"
            >
              <Upload className="w-3.5 h-3.5 text-slate-500" />
              <span>Import</span>
            </button>

            {/* Export Button */}
            <a
              href="/api/hostlist/export"
              download
              className="p-1.5 bg-white hover:bg-slate-100 border border-slate-200 rounded-lg text-slate-600 hover:text-slate-800 transition-colors flex items-center gap-1.5 text-xs font-semibold"
            >
              <Download className="w-3.5 h-3.5 text-slate-500" />
              <span>Export CSV</span>
            </a>

            {/* Add Host Primary Button */}
            <button
              onClick={() => {
                setEditingHost(null);
                setFormMode("add");
              }}
              className="p-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center gap-1.5 text-xs font-semibold shadow-sm shadow-blue-500/10 hover:shadow"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>Register Host</span>
            </button>
          </div>
        </div>
      </header>

      {/* Main Panel Content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Central Workspace */}
        <main className="flex-1 flex flex-col overflow-hidden min-w-0">
          {/* Upper Quick Metric Badges (Interactive) */}
          <section className="p-4 flex flex-wrap gap-2 shrink-0 border-b border-slate-100 bg-white">
            {CATEGORIES.map((cat) => {
              const isActive = selectedCategory === cat.name;
              const count = getCategoryCount(cat.name);
              const colorClass = cat.color;

              // Design aesthetic: Pill shape
              const activeClass = isActive 
                ? "bg-slate-900 text-white border-slate-900 shadow-sm"
                : "bg-white text-slate-600 border-slate-200 hover:bg-slate-50";

              return (
                <button
                  key={cat.name}
                  onClick={() => {
                    setSelectedCategory(cat.name);
                    setSelectedTag(null);
                  }}
                  className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold border transition-all cursor-pointer ${activeClass}`}
                >
                  <span className={`w-1.5 h-1.5 rounded-full ${isActive ? "bg-white" : `${colorClass} bg-current`}`}></span>
                  <span>{cat.label}</span>
                  <span className={`font-mono text-[10px] px-1.5 py-0.2 rounded-full ${isActive ? "bg-slate-800 text-slate-200" : "bg-slate-100 text-slate-500"}`}>
                    {count}
                  </span>
                </button>
              );
            })}
          </section>

          {/* Recursive Tag Filters */}
          {selectedCategory !== "All" && availableTags.length > 0 && (
            <section className="px-4 py-2 flex flex-wrap items-center gap-2 border-b border-slate-100 bg-slate-50/50 shrink-0">
              <span className="text-[10px] font-bold text-slate-400 uppercase tracking-wider mr-1">
                Filter by Tag:
              </span>
              <button
                onClick={() => setSelectedTag(null)}
                className={`inline-flex items-center px-2 py-0.5 rounded-full text-[11px] font-semibold border transition-all cursor-pointer ${
                  !selectedTag
                    ? "bg-blue-600 text-white border-blue-600 shadow-sm"
                    : "bg-white text-slate-600 border-slate-200 hover:bg-slate-50"
                }`}
              >
                All Tags
              </button>
              {availableTags.map((tag) => {
                const isTagActive = selectedTag === tag;
                return (
                  <button
                    key={tag}
                    onClick={() => setSelectedTag(isTagActive ? null : tag)}
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-medium border transition-all cursor-pointer ${
                      isTagActive
                        ? "bg-slate-900 text-white border-slate-900 shadow-sm"
                        : "bg-blue-50 text-blue-700 border-blue-100 hover:bg-blue-100"
                    }`}
                  >
                    #{tag}
                  </button>
                );
              })}
            </section>
          )}

          {/* Filtering, Search & Settings Control Panel */}
          <section className="bg-white border-y border-slate-200 px-4 py-2.5 flex flex-col md:flex-row items-center justify-between gap-3 shrink-0">
            {/* Search inputs */}
            <div className="relative w-full md:max-w-md">
              <div className="absolute inset-y-0 left-3 flex items-center pointer-events-none text-slate-400">
                <Search className="w-4 h-4" />
              </div>
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search hostname, IP, username, tags..."
                className="w-full text-xs pl-9 pr-8 py-2 bg-slate-50 border border-slate-200 rounded-lg text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery("")}
                  className="absolute inset-y-0 right-2.5 flex items-center text-slate-400 hover:text-slate-600"
                >
                  <X className="w-3.5 h-3.5" />
                </button>
              )}
            </div>

            {/* Layout Density Controls & Info */}
            <div className="flex items-center justify-between w-full md:w-auto gap-4">
              {/* Category selector on mobile */}
              <div className="block lg:hidden text-xs">
                <select
                  value={selectedCategory}
                  onChange={(e) => {
                    setSelectedCategory(e.target.value);
                    setSelectedTag(null);
                  }}
                  className="p-1.5 bg-white border border-slate-200 rounded-lg text-slate-700 focus:outline-none"
                >
                  {CATEGORIES.map((cat) => (
                    <option key={cat.name} value={cat.name}>
                      {cat.label}
                    </option>
                  ))}
                </select>
              </div>

              {/* Table Density Selector */}
              <div className="flex items-center gap-1">
                <span className="text-[10px] font-bold text-slate-400 uppercase tracking-tight mr-1">
                  Density:
                </span>
                <div className="inline-flex bg-slate-100 p-0.5 rounded-lg border border-slate-200">
                  {(["normal", "dense", "super-dense"] as TableDensity[]).map((d) => (
                    <button
                      key={d}
                      onClick={() => setDensity(d)}
                      className={`text-[10px] px-2 py-1 rounded font-semibold capitalize transition-all ${
                        density === d
                          ? "bg-white text-slate-800 shadow-sm"
                          : "text-slate-500 hover:text-slate-800"
                      }`}
                    >
                      {d.replace("-", " ")}
                    </button>
                  ))}
                </div>
              </div>

              <div className="text-[11px] text-slate-400 font-mono hidden xl:block">
                Showing <strong className="text-slate-700">{sortedHostList.length}</strong> / {hostList.length}
              </div>
            </div>
          </section>

          {/* High Density Credentials Table */}
          <div className="flex-1 overflow-auto bg-white relative">
            {loading ? (
              <div className="absolute inset-0 flex flex-col items-center justify-center bg-white/80 z-10 gap-3">
                <RefreshCw className="w-8 h-8 text-blue-600 animate-spin" />
                <span className="text-xs font-semibold text-slate-500">Loading hosts...</span>
              </div>
            ) : sortedHostList.length === 0 ? (
              <div className="p-12 text-center">
                <div className="w-12 h-12 bg-slate-100 rounded-full flex items-center justify-center mx-auto text-slate-400 mb-3">
                  <Search className="w-6 h-6" />
                </div>
                <h3 className="text-sm font-bold text-slate-800">No matching hosts found</h3>
                <p className="text-xs text-slate-500 mt-1 max-w-sm mx-auto">
                  No registered servers match your search or category filters. Check spelling or clear filters.
                </p>
                {(searchQuery || selectedCategory !== "All" || selectedTag) && (
                  <button
                    onClick={() => {
                      setSearchQuery("");
                      setSelectedCategory("All");
                      setSelectedTag(null);
                    }}
                    className="mt-3 px-3 py-1 text-xs text-blue-600 bg-blue-50 border border-blue-200 hover:bg-blue-100 rounded-lg transition-all"
                  >
                    Reset Filter Search
                  </button>
                )}
              </div>
            ) : (
              <table className="w-full text-left border-collapse table-fixed min-w-[900px]">
                {/* Headers with Sorting */}
                <thead className="bg-slate-50 text-[10px] font-bold text-slate-400 uppercase tracking-wider sticky top-0 z-10 border-b border-slate-200">
                  <tr>
                    <th 
                      onClick={() => handleSort("platform")}
                      className={`cursor-pointer hover:bg-slate-100 select-none pl-4 pr-2 font-bold ${
                        density === "super-dense" ? "py-1.5 w-[110px]" : density === "dense" ? "py-2 w-[130px]" : "py-3 w-[150px]"
                      }`}
                    >
                      <div className="flex items-center gap-1 justify-between">
                        <span>Platform</span>
                        <ArrowUpDown className="w-3 h-3 text-slate-300" />
                      </div>
                    </th>
                    
                    <th 
                      onClick={() => handleSort("hostname")}
                      className={`cursor-pointer hover:bg-slate-100 select-none px-2 font-bold ${
                        density === "super-dense" ? "py-1.5 w-[180px]" : density === "dense" ? "py-2 w-[220px]" : "py-3 w-[260px]"
                      }`}
                    >
                      <div className="flex items-center gap-1 justify-between">
                        <span>Hostname / Endpoint</span>
                        <ArrowUpDown className="w-3 h-3 text-slate-300" />
                      </div>
                    </th>

                    <th 
                      onClick={() => handleSort("ip")}
                      className={`cursor-pointer hover:bg-slate-100 select-none px-2 font-bold ${
                        density === "super-dense" ? "py-1.5 w-[150px]" : density === "dense" ? "py-2 w-[180px]" : "py-3 w-[210px]"
                      }`}
                    >
                      <div className="flex items-center gap-1 justify-between">
                        <span>IP Address (Port)</span>
                        <ArrowUpDown className="w-3 h-3 text-slate-300" />
                      </div>
                    </th>

                    <th 
                      className={`px-2 font-bold ${
                        density === "super-dense" ? "py-1.5 w-[240px]" : density === "dense" ? "py-2 w-[280px]" : "py-3 w-[320px]"
                      }`}
                    >
                      <div className="flex items-center gap-1">
                        <span>USERLIST</span>
                        <Lock className="w-3 h-3 text-slate-300" />
                      </div>
                    </th>

                    <th 
                      className={`px-2 font-bold ${
                        density === "super-dense" ? "py-1.5" : density === "dense" ? "py-2" : "py-3"
                      }`}
                    >
                      <span>Tags & Notes</span>
                    </th>

                    <th 
                      className={`px-4 text-right font-bold ${
                        density === "super-dense" ? "py-1.5 w-[70px]" : density === "dense" ? "py-2 w-[90px]" : "py-3 w-[110px]"
                      }`}
                    >
                      <span>Actions</span>
                    </th>
                  </tr>
                </thead>

                {/* Body Rows */}
                <tbody className="divide-y divide-slate-100 text-slate-700">
                  {sortedHostList.map((h) => {
                    const PlatformIcon = platformIcons[h.platform] || Server;
                    const badgeClass = platformBadgeColors[h.platform] || platformBadgeColors.Other;
                    
                    const isCopiedHostname = !!copiedStates[`${h.id}-hostname`];
                    const isCopiedIP = !!copiedStates[`${h.id}-ip`];

                    const rowPadding = 
                      density === "super-dense" 
                        ? "py-0.5 px-2 text-[11px]" 
                        : density === "dense" 
                        ? "py-1.5 px-2 text-xs" 
                        : "py-3 px-2 text-sm";

                    const rowHeight = density === "super-dense" ? "h-6" : density === "dense" ? "h-10" : "h-14";

                    return (
                      <tr 
                        key={h.id} 
                        className={`hover:bg-slate-50 transition-colors ${rowHeight} group`}
                      >
                        {/* Platform Badge Column */}
                        <td className={`${rowPadding} pl-4`}>
                          <div className="flex items-center gap-1.5 truncate">
                            <span className={`p-1 rounded-md shrink-0 ${badgeClass}`}>
                              <PlatformIcon className="w-3.5 h-3.5" />
                            </span>
                            <span className="font-semibold text-slate-800 truncate">
                              {h.platform}
                            </span>
                          </div>
                        </td>

                        {/* Hostname Column */}
                        <td className={`${rowPadding} font-mono font-medium text-slate-900 group/host relative`}>
                          <div className="flex items-center justify-between gap-1.5 pr-2">
                            <span 
                              title={h.hostname}
                              className="truncate select-all cursor-pointer hover:text-blue-600 transition-colors"
                              onClick={() => handleCopyToClipboard(h.hostname, `${h.id}-hostname`, "Hostname")}
                            >
                              {h.hostname}
                            </span>
                            <button
                              onClick={() => handleCopyToClipboard(h.hostname, `${h.id}-hostname`, "Hostname")}
                              className="opacity-0 group-hover/host:opacity-100 p-0.5 hover:bg-slate-200 text-slate-400 hover:text-slate-700 rounded transition-opacity"
                              title="Copy Hostname"
                            >
                              {isCopiedHostname ? (
                                <Check className="w-3 h-3 text-emerald-600" />
                              ) : (
                                <Copy className="w-3 h-3" />
                              )}
                            </button>
                          </div>
                        </td>

                        {/* IP Address and Port Column */}
                        <td className={`${rowPadding} font-mono text-slate-500 group/ip whitespace-nowrap`}>
                          <div className="flex items-center justify-between gap-1 pr-2">
                            <div className="flex items-center min-w-0">
                              <span 
                                title={h.ip}
                                className="cursor-pointer hover:text-blue-600"
                                onClick={() => handleCopyToClipboard(h.ip, `${h.id}-ip`, "IP Address")}
                              >
                                {h.ip || "---"}
                              </span>
                              {h.port && (
                                <span className="text-[10px] text-slate-400 font-normal ml-1">
                                  :{h.port}
                                </span>
                              )}
                            </div>
                            {h.ip && (
                              <button
                                onClick={() => handleCopyToClipboard(h.ip, `${h.id}-ip`, "IP Address")}
                                className="opacity-0 group-hover/ip:opacity-100 p-0.5 hover:bg-slate-200 text-slate-400 hover:text-slate-700 rounded transition-opacity"
                                title="Copy IP"
                              >
                                {isCopiedIP ? (
                                  <Check className="w-3 h-3 text-emerald-600" />
                                ) : (
                                  <Copy className="w-3 h-3" />
                                )}
                              </button>
                            )}
                          </div>
                        </td>

                        {/* USERLIST Column */}
                        <td className={`${rowPadding}`}>
                          <div className="flex flex-wrap gap-1.5 max-w-full">
                            {h.userlist && h.userlist.map((user: any, idx: number) => {
                              const key = `${h.id}-${idx}`;
                              const isPassShown = !!revealedPasswords[key];
                              const isCopiedUser = !!copiedStates[`${key}-user`];
                              const isCopiedPass = !!copiedStates[`${key}-pass`];
                              return (
                                <div 
                                  key={idx} 
                                  className="inline-flex items-center gap-1 bg-slate-50 border border-slate-200 rounded-lg px-2 py-0.5 text-xs text-slate-700 font-medium whitespace-nowrap shadow-sm hover:bg-slate-100 transition-all"
                                >
                                  {isPassShown ? (
                                    <>
                                      <span className="font-mono text-[10px] text-blue-700 font-semibold select-all animate-fade-in">
                                        {user.password}
                                      </span>
                                      <span className="text-slate-300">|</span>
                                    </>
                                  ) : (
                                    <>
                                      <span className="font-semibold">{user.username}</span>
                                      <span className="text-slate-300">|</span>
                                    </>
                                  )}
                                  <button
                                    onClick={() => {
                                      setRevealedPasswords(prev => {
                                        const next = { ...prev };
                                        if (next[key]) {
                                          delete next[key];
                                        } else {
                                          next[key] = true;
                                          // Auto-mask after 15 seconds
                                          setTimeout(() => {
                                            setRevealedPasswords(current => {
                                              const updated = { ...current };
                                              delete updated[key];
                                              return updated;
                                            });
                                          }, 15000);
                                        }
                                        return next;
                                      });
                                    }}
                                    className="p-0.5 hover:bg-slate-200 rounded text-slate-400 hover:text-slate-700 transition-all"
                                    title={isPassShown ? "Mask Password & Show Username" : "Reveal Password for 15s"}
                                  >
                                    {isPassShown ? (
                                      <EyeOff className="w-3 h-3 text-blue-600" />
                                    ) : (
                                      <Eye className="w-3 h-3" />
                                    )}
                                  </button>
                                  <div className="flex items-center gap-0.5 border-l border-slate-200 pl-1">
                                    <button
                                      onClick={() => handleCopyToClipboard(user.username, `${key}-user`, "Username")}
                                      className="p-0.5 hover:bg-slate-200 rounded text-slate-400 hover:text-slate-600 transition-all"
                                      title="Copy Username"
                                    >
                                      {isCopiedUser ? (
                                        <Check className="w-3 h-3 text-emerald-600" />
                                      ) : (
                                        <Copy className="w-3 h-3" />
                                      )}
                                    </button>
                                    <button
                                      onClick={() => handleCopyToClipboard(user.password, `${key}-pass`, "Password")}
                                      className="p-0.5 hover:bg-slate-200 rounded text-slate-400 hover:text-slate-600 transition-all"
                                      title="Copy Password"
                                    >
                                      {isCopiedPass ? (
                                        <Check className="w-3 h-3 text-emerald-600" />
                                      ) : (
                                        <Key className="w-3 h-3" />
                                      )}
                                    </button>
                                  </div>
                                </div>
                              );
                            })}
                          </div>
                        </td>

                        {/* Tags and Description Columns (Integrated for high density) */}
                        <td className={`${rowPadding} max-w-xs`}>
                          <div className="flex flex-col gap-0.5 max-w-full">
                            {/* Tags row */}
                            {h.tags && (
                              <div className="flex flex-wrap gap-1 max-h-5 overflow-hidden">
                                {h.tags.split(",").map((tag) => (
                                  <button
                                    key={tag}
                                    onClick={() => setSearchQuery(tag.trim())}
                                    className="text-[9px] font-bold px-1.5 py-0.2 bg-slate-100 text-slate-600 border border-slate-200 rounded-full hover:bg-blue-50 hover:text-blue-600 hover:border-blue-200 transition-all truncate shrink-0"
                                  >
                                    #{tag.trim()}
                                  </button>
                                ))}
                              </div>
                            )}

                            {/* Description subtext */}
                            {h.description ? (
                              <p 
                                title={h.description}
                                className="text-[10px] text-slate-400 truncate max-w-full leading-normal"
                              >
                                {h.description}
                              </p>
                            ) : (
                              !h.tags && <span className="text-slate-300 italic text-[10px]">no description or tags</span>
                            )}
                          </div>
                        </td>

                        {/* Action buttons */}
                        <td className={`${rowPadding} text-right pr-4 shrink-0`}>
                          <div className="flex items-center justify-end gap-1 opacity-100 sm:opacity-0 group-hover:opacity-100 transition-opacity">
                            <button
                              onClick={() => {
                                setEditingHost(h);
                                setFormMode("edit");
                              }}
                              className="p-1 hover:bg-amber-50 text-slate-400 hover:text-amber-600 border border-transparent hover:border-amber-200 rounded-md transition-all"
                              title="Edit Host Details"
                            >
                              <Edit2 className="w-3 h-3" />
                            </button>
                            <button
                              onClick={() => setDeletingId(h.id)}
                              className="p-1 hover:bg-rose-50 text-slate-400 hover:text-rose-600 border border-transparent hover:border-rose-200 rounded-md transition-all"
                              title="Delete Host"
                            >
                              <Trash2 className="w-3 h-3" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>
        </main>
      </div>

      {/* Slide-out Drawer Form Panel */}
      {formMode !== "closed" && (
        <div className="fixed inset-0 bg-black/40 z-50 flex animate-fade-in">
          <div className="flex-1" onClick={() => setFormMode("closed")}></div>
          <CredentialForm
            credential={formMode === "edit" ? editingHost : null}
            onSave={handleSave}
            onCancel={() => {
              setFormMode("closed");
              setEditingHost(null);
            }}
          />
        </div>
      )}

      {/* Delete Confirmation Modal Dialog */}
      {deletingId !== null && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50 animate-fade-in">
          <div className="bg-white rounded-xl shadow-2xl max-w-sm w-full p-5 border border-slate-200">
            <div className="flex items-center gap-3 text-rose-600 mb-3">
              <div className="p-2 bg-rose-50 rounded-full">
                <Trash2 className="w-5 h-5" />
              </div>
              <h3 className="font-bold text-sm text-slate-800">Confirm Deletion</h3>
            </div>
            <p className="text-xs text-slate-500 leading-relaxed mb-4">
              Are you absolutely sure you want to delete this host? This action writes immediately back to the local database files and is irreversible.
            </p>
            <div className="flex justify-end gap-2 text-xs">
              <button
                onClick={() => setDeletingId(null)}
                className="px-3.5 py-1.5 font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => handleDelete(deletingId)}
                className="px-4 py-1.5 font-semibold text-white bg-rose-600 hover:bg-rose-700 rounded-lg shadow-sm transition-all"
              >
                Confirm Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Import CSV Modal Dialog */}
      {isImportOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50 animate-fade-in">
          <div className="bg-white rounded-xl shadow-2xl max-w-md w-full p-5 border border-slate-200">
            <div className="flex items-center justify-between border-b border-slate-100 pb-3 mb-4">
              <div className="flex items-center gap-2 text-slate-800">
                <div className="p-1.5 bg-blue-50 text-blue-600 rounded-lg">
                  <Upload className="w-4 h-4" />
                </div>
                <h3 className="font-bold text-sm">Bulk Import from CSV</h3>
              </div>
              <button
                onClick={() => setIsImportOpen(false)}
                className="p-1 hover:bg-slate-100 rounded-md text-slate-400 hover:text-slate-600"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-4 text-xs">
              <p className="text-slate-500 leading-relaxed">
                Choose a `.csv` file with headers matching the Host List schema (without credentials): 
                <code className="font-mono bg-slate-100 px-1 py-0.5 rounded text-[10px] ml-1">
                  hostname, ip, platform, port, tags, description
                </code>
              </p>

              {/* Import Options */}
              <div className="bg-slate-50 border border-slate-200 rounded-xl p-3 space-y-2">
                <span className="font-bold text-slate-600 block text-[10px] uppercase tracking-wider">
                  Import Conflict Resolution:
                </span>
                <label className="flex items-start gap-2.5 cursor-pointer text-slate-700 hover:text-slate-900">
                  <input
                    type="radio"
                    name="importOption"
                    checked={importMergeOption === "merge"}
                    onChange={() => setImportMergeOption("merge")}
                    className="mt-0.5 text-blue-600 focus:ring-blue-500 w-3.5 h-3.5"
                  />
                  <div>
                    <span className="font-bold block">Merge and Update (Recommended)</span>
                    <span className="text-[10px] text-slate-400">Updates existing hosts if they match on hostname, leaving other servers untouched.</span>
                  </div>
                </label>

                <label className="flex items-start gap-2.5 cursor-pointer text-slate-700 hover:text-slate-900 mt-2">
                  <input
                    type="radio"
                    name="importOption"
                    checked={importMergeOption === "overwrite"}
                    onChange={() => setImportMergeOption("overwrite")}
                    className="mt-0.5 text-rose-600 focus:ring-rose-500 w-3.5 h-3.5"
                  />
                  <div>
                    <span className="font-bold text-rose-700 block">Overwrite Database</span>
                    <span className="text-[10px] text-slate-400">Completely replaces current CSV database with the uploaded file data. Use with caution.</span>
                  </div>
                </label>
              </div>

              {/* Hidden file input */}
              <input
                type="file"
                ref={fileInputRef}
                onChange={handleCSVUpload}
                accept=".csv"
                className="hidden"
              />

              <div className="flex justify-end gap-2 pt-2">
                <button
                  onClick={() => setIsImportOpen(false)}
                  className="px-3.5 py-1.5 font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 rounded-lg transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={triggerImport}
                  className="px-4 py-1.5 font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-all flex items-center gap-1.5"
                >
                  <Upload className="w-3.5 h-3.5" />
                  Select CSV File & Execute
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
