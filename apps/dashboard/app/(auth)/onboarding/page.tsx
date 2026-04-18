"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { createApiClient } from "@/lib/api-client";
import { Megaphone, MonitorPlay, CheckCircle2 } from "lucide-react";

type OrgType = "publisher" | "brand";

const STEPS = ["Choose type", "Name your org", "All set"] as const;

const orgDetailsSchema = z.object({
  name: z.string().min(2, "Name must be at least 2 characters"),
  slug: z
    .string()
    .min(2, "Slug must be at least 2 characters")
    .max(50, "Slug must be at most 50 characters")
    .regex(
      /^[a-z0-9-]+$/,
      "Slug can only contain lowercase letters, numbers and hyphens"
    ),
});

type OrgDetailsFormValues = z.infer<typeof orgDetailsSchema>;

function slugify(value: string): string {
  return value
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]/g, "")
    .replace(/-+/g, "-")
    .slice(0, 50);
}

interface StepIndicatorProps {
  current: number;
  total: number;
}

function StepIndicator({ current, total }: StepIndicatorProps) {
  return (
    <div className="flex items-center gap-2">
      {Array.from({ length: total }).map((_, i) => (
        <div
          key={i}
          className={cn(
            "h-2 rounded-full transition-all",
            i < current
              ? "w-6 bg-primary"
              : i === current
                ? "w-6 bg-primary"
                : "w-2 bg-muted"
          )}
        />
      ))}
    </div>
  );
}

interface OrgTypeCardProps {
  type: OrgType;
  selected: boolean;
  onSelect: (type: OrgType) => void;
}

function OrgTypeCard({ type, selected, onSelect }: OrgTypeCardProps) {
  const isPublisher = type === "publisher";
  const Icon = isPublisher ? MonitorPlay : Megaphone;
  const title = isPublisher ? "Publisher" : "Brand";
  const description = isPublisher
    ? "Monetize your apps and content with targeted ads"
    : "Run advertising campaigns to reach your audience";

  return (
    <button
      type="button"
      onClick={() => onSelect(type)}
      className={cn(
        "flex flex-col items-start gap-3 rounded-xl border-2 p-5 text-left transition-all hover:border-primary/60",
        selected
          ? "border-primary bg-primary/5"
          : "border-border bg-card"
      )}
    >
      <div
        className={cn(
          "flex h-10 w-10 items-center justify-center rounded-lg",
          selected ? "bg-primary text-primary-foreground" : "bg-muted"
        )}
      >
        <Icon className="h-5 w-5" />
      </div>
      <div>
        <p className="font-semibold">{title}</p>
        <p className="text-sm text-muted-foreground mt-0.5">{description}</p>
      </div>
    </button>
  );
}

export default function OnboardingPage() {
  const router = useRouter();
  const [step, setStep] = useState(0);
  const [orgType, setOrgType] = useState<OrgType | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<OrgDetailsFormValues>({
    resolver: zodResolver(orgDetailsSchema),
    defaultValues: { name: "", slug: "" },
  });

  const nameValue = watch("name");

  const handleNameChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const name = e.target.value;
      setValue("name", name);
      // Auto-generate slug from name only if slug hasn't been manually edited
      setValue("slug", slugify(name), { shouldValidate: false });
    },
    [setValue]
  );

  function handleTypeSelect(type: OrgType) {
    setOrgType(type);
  }

  function handleNextFromStep1() {
    if (!orgType) {
      toast.error("Please select an organization type");
      return;
    }
    setStep(1);
  }

  async function onSubmitDetails(values: OrgDetailsFormValues) {
    if (!orgType) return;

    setIsSubmitting(true);
    try {
      const client = createApiClient();
      const { data, error } = await client.POST("/v1/organizations", {
        body: {
          type: orgType,
          name: values.name,
          slug: values.slug,
        },
      });

      if (error) {
        toast.error("Failed to create organization. Please try again.");
        return;
      }

      if (data) {
        toast.success("Organization created!");
        setStep(2);
      }
    } catch {
      toast.error("Something went wrong. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  }

  function handleFinish() {
    if (orgType === "publisher") {
      router.push("/apps");
    } else {
      router.push("/campaigns");
    }
    router.refresh();
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/20 px-4">
      <div className="w-full max-w-lg space-y-6">
        {/* Header */}
        <div className="space-y-1">
          <p className="text-sm text-muted-foreground font-medium">
            Step {step + 1} of {STEPS.length} — {STEPS[step]}
          </p>
          <StepIndicator current={step} total={STEPS.length} />
        </div>

        {/* Step 1: Org type */}
        {step === 0 && (
          <Card>
            <CardHeader>
              <CardTitle>What best describes you?</CardTitle>
              <CardDescription>
                This helps us tailor your experience on the platform.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <OrgTypeCard
                  type="publisher"
                  selected={orgType === "publisher"}
                  onSelect={handleTypeSelect}
                />
                <OrgTypeCard
                  type="brand"
                  selected={orgType === "brand"}
                  onSelect={handleTypeSelect}
                />
              </div>
              <Button
                className="w-full"
                onClick={handleNextFromStep1}
                disabled={!orgType}
              >
                Continue
              </Button>
            </CardContent>
          </Card>
        )}

        {/* Step 2: Org name + slug */}
        {step === 1 && (
          <Card>
            <CardHeader>
              <CardTitle>Name your organization</CardTitle>
              <CardDescription>
                You can change this later in your settings.
              </CardDescription>
            </CardHeader>
            <form onSubmit={handleSubmit(onSubmitDetails)}>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="org-name">Organization name</Label>
                  <Input
                    id="org-name"
                    placeholder="Acme Corp"
                    disabled={isSubmitting}
                    {...register("name")}
                    onChange={handleNameChange}
                  />
                  {errors.name && (
                    <p className="text-xs text-destructive">
                      {errors.name.message}
                    </p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="org-slug">
                    Slug{" "}
                    <span className="text-muted-foreground font-normal">
                      (URL-friendly name)
                    </span>
                  </Label>
                  <Input
                    id="org-slug"
                    placeholder="acme-corp"
                    disabled={isSubmitting}
                    {...register("slug")}
                  />
                  {errors.slug && (
                    <p className="text-xs text-destructive">
                      {errors.slug.message}
                    </p>
                  )}
                  {nameValue && (
                    <p className="text-xs text-muted-foreground">
                      Your org URL: /orgs/{watch("slug")}
                    </p>
                  )}
                </div>
                <div className="flex gap-3">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setStep(0)}
                    disabled={isSubmitting}
                  >
                    Back
                  </Button>
                  <Button
                    type="submit"
                    className="flex-1"
                    disabled={isSubmitting}
                  >
                    {isSubmitting ? "Creating..." : "Create organization"}
                  </Button>
                </div>
              </CardContent>
            </form>
          </Card>
        )}

        {/* Step 3: Success */}
        {step === 2 && (
          <Card>
            <CardHeader>
              <div className="flex justify-center mb-2">
                <CheckCircle2 className="h-12 w-12 text-green-500" />
              </div>
              <CardTitle className="text-center">You&apos;re all set!</CardTitle>
              <CardDescription className="text-center">
                Your organization has been created. Time to get started.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button className="w-full" onClick={handleFinish}>
                {orgType === "publisher"
                  ? "Go to Apps"
                  : "Go to Campaigns"}
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
