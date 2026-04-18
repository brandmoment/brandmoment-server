import { redirect } from "next/navigation";
import { auth } from "@/lib/auth";
import { headers } from "next/headers";

export default async function DashboardHome() {
  const session = await auth.api.getSession({
    headers: await headers(),
  });

  if (!session) {
    redirect("/login");
  }

  const activeOrg = session.session.activeOrganizationId
    ? await auth.api
        .getFullOrganization({
          headers: await headers(),
          query: { organizationId: session.session.activeOrganizationId },
        })
        .catch(() => null)
    : null;

  if (!activeOrg) {
    redirect("/onboarding");
  }

  const orgType = activeOrg.metadata?.type as string | undefined;

  if (orgType === "publisher") {
    redirect("/apps");
  }

  if (orgType === "brand") {
    redirect("/campaigns");
  }

  if (orgType === "admin") {
    redirect("/admin/organizations");
  }

  // Fallback: onboarding not complete
  redirect("/onboarding");
}
