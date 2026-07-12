/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

export interface HostList {
  id: string;
  hostname: string;
  ip: string;
  platform: string;
  port: string;
  tags: string;
  description: string;
  updatedAt: string;
  userlist: { username: string; password: string }[];
}

export type TableDensity = "normal" | "dense" | "super-dense";
