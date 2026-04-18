interface AcceptInvitePageProps {
  params: Promise<{ token: string }>;
}

export default async function AcceptInvitePage({
  params,
}: AcceptInvitePageProps) {
  const { token } = await params;

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/20 px-4">
      <div className="w-full max-w-md space-y-4 text-center">
        <div className="space-y-2">
          <h1 className="text-2xl font-bold tracking-tight">
            Invite Acceptance
          </h1>
          <p className="text-muted-foreground">
            Invite acceptance is being set up and will be available soon.
          </p>
        </div>
        <div className="rounded-lg border bg-muted/40 px-4 py-3 text-sm">
          <span className="font-medium text-muted-foreground">
            Your invite token:{" "}
          </span>
          <code className="font-mono text-xs break-all">{token}</code>
        </div>
      </div>
    </div>
  );
}
