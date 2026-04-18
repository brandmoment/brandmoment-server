export type OrgRole = "owner" | "admin" | "editor" | "viewer";

export type OrgType = "admin" | "publisher" | "brand";

export interface OrgMembership {
  org_id: string;
  role: OrgRole;
}

export interface Organization {
  id: string;
  type: OrgType;
  name: string;
  slug: string;
  created_at: string;
  updated_at: string;
}

export interface UserProfile {
  id: string;
  email: string;
  name: string;
  created_at: string;
  orgs: OrgMembership[];
}
