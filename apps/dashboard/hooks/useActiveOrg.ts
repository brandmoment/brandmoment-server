"use client";

import { useContext } from "react";
import { OrgContext } from "@/app/providers";

export function useActiveOrg() {
  const ctx = useContext(OrgContext);
  if (!ctx) {
    throw new Error("useActiveOrg must be used within OrgProvider");
  }
  return ctx;
}
