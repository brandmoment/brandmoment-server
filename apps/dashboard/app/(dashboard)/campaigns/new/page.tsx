"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

// Direct navigation to /campaigns/new bounces to /campaigns.
// The create dialog is initiated from the list page.
export default function NewCampaignPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/campaigns");
  }, [router]);

  return null;
}
