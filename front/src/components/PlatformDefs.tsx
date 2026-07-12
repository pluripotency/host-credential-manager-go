import { 
  Terminal, Cpu, Network, Layers, Server, Database, Shield, Activity 
} from "lucide-react";
import type { HostList } from "../types";

// Map platforms to specific icon components
export const platformIcons: Record<string, any> = {
  Linux: Terminal,
  Windows: Cpu,
  macOS: Cpu,
  Cisco: Network,
  AWS: Layers,
  VMware: Server,
  Proxmox: Server,
  MySQL: Database,
  PostgreSQL: Database,
  Redis: Database,
  MongoDB: Database,
  Oracle: Database,
  Kubernetes: Activity,
  FreeNAS: Server,
  OPNsense: Shield,
  FortiGate: Shield,
  Grafana: Activity,
  Other: Server,
};

// Map platforms to color badge classes
export const platformBadgeColors: Record<string, string> = {
  Linux: "bg-emerald-50 text-emerald-700 border-emerald-100",
  Windows: "bg-blue-50 text-blue-700 border-blue-100",
  macOS: "bg-slate-100 text-slate-700 border-slate-200",
  Cisco: "bg-amber-50 text-amber-700 border-amber-100",
  AWS: "bg-orange-50 text-orange-700 border-orange-100",
  VMware: "bg-indigo-50 text-indigo-700 border-indigo-100",
  Proxmox: "bg-sky-50 text-sky-700 border-sky-100",
  MySQL: "bg-violet-50 text-violet-700 border-violet-100",
  PostgreSQL: "bg-purple-50 text-purple-700 border-purple-100",
  Redis: "bg-rose-50 text-rose-700 border-rose-100",
  MongoDB: "bg-green-50 text-green-700 border-green-100",
  Oracle: "bg-red-50 text-red-700 border-red-100",
  Kubernetes: "bg-cyan-50 text-cyan-700 border-cyan-100",
  FreeNAS: "bg-teal-50 text-teal-700 border-teal-100",
  OPNsense: "bg-pink-50 text-pink-700 border-pink-100",
  FortiGate: "bg-pink-50 text-pink-700 border-pink-100",
  Grafana: "bg-fuchsia-50 text-fuchsia-700 border-fuchsia-100",
  Other: "bg-slate-100 text-slate-700 border-slate-200",
};

// Platform options for selection
export interface PlatformOption {
  value: string;
  label: string;
}

export const platformOptions: PlatformOption[] = [
  { value: "Linux", label: "Linux (SSH)" },
  { value: "Windows", label: "Windows (RDP)" },
  { value: "macOS", label: "macOS" },
  { value: "AWS", label: "AWS EC2 / RDS" },
  { value: "Cisco", label: "Cisco IOS" },
  { value: "VMware", label: "VMware ESXi / vCenter" },
  { value: "Proxmox", label: "Proxmox VE" },
  { value: "MySQL", label: "MySQL Server" },
  { value: "PostgreSQL", label: "PostgreSQL Server" },
  { value: "Redis", label: "Redis Database" },
  { value: "MongoDB", label: "MongoDB Cluster" },
  { value: "Oracle", label: "Oracle DB" },
  { value: "Kubernetes", label: "Kubernetes Cluster" },
  { value: "FreeNAS", label: "TrueNAS / FreeNAS" },
  { value: "OPNsense", label: "OPNsense / pfSense" },
  { value: "FortiGate", label: "FortiGate" },
  { value: "Grafana", label: "Grafana" },
  { value: "Other", label: "Other Platform" },
];

// Default ports mapping
export const PLATFORM_PORTS: Record<string, string> = {
  Linux: "22",
  Windows: "3389",
  macOS: "22",
  Cisco: "22",
  AWS: "443",
  VMware: "443",
  MySQL: "3306",
  PostgreSQL: "5432",
  Redis: "6379",
  MongoDB: "27017",
  Kubernetes: "6443",
  Proxmox: "8006",
  Grafana: "3000",
};

// Category definitions matching credentials.filter and metrics counting in App.tsx
export interface CategoryDef {
  name: string;
  label: string;
  color: string;
  match: (cred: HostList) => boolean;
}

export const CATEGORIES: CategoryDef[] = [
  {
    name: "All",
    label: "All Hosts",
    color: "text-blue-600",
    match: () => true,
  },
  {
    name: "Linux",
    label: "Linux Nodes",
    color: "text-emerald-600",
    match: (cred) => cred.platform === "Linux",
  },
  {
    name: "Windows",
    label: "Windows Domain",
    color: "text-blue-500",
    match: (cred) => cred.platform === "Windows",
  },
  {
    name: "Database",
    label: "Database Instances",
    color: "text-violet-600",
    match: (cred) =>
      ["MySQL", "PostgreSQL", "Redis", "MongoDB", "Oracle"].includes(cred.platform) ||
      cred.tags.includes("database"),
  },
  {
    name: "Network",
    label: "Network Devices",
    color: "text-amber-600",
    match: (cred) =>
      ["Cisco", "OPNsense", "FortiGate", "network"].includes(cred.platform) ||
      cred.tags.includes("network"),
  },
  {
    name: "Virtualization",
    label: "VMware / Proxmox",
    color: "text-sky-600",
    match: (cred) =>
      ["VMware", "Proxmox", "Kubernetes"].includes(cred.platform) ||
      cred.tags.includes("virtualization"),
  },
  {
    name: "Production",
    label: "Production Tier",
    color: "text-rose-600",
    match: (cred) => cred.tags.includes("production"),
  },
  {
    name: "Staging",
    label: "Staging Tier",
    color: "text-purple-600",
    match: (cred) => cred.tags.includes("staging"),
  },
];
