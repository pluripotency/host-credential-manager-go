/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

export interface AccessItem {
  protocol: string;
  port: string;
  path?: string;
}

export interface HostList {
  id: string;
  hostname: string;
  ip: string;
  platform: string;
  os?: string;
  port: string;
  tags: string;
  description: string;
  updatedAt: string;
  userlist: { username: string; password: string }[];
  accesslist?: AccessItem[];
}

export type TableDensity = "normal" | "dense" | "super-dense";
