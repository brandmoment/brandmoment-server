"use client";

import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { useCreateCampaign } from "@/hooks/useCreateCampaign";

const createCampaignSchema = z
  .object({
    name: z
      .string()
      .min(1, "Name is required")
      .max(200, "Name must be at most 200 characters"),
    budget_cents: z.string().optional(),
    currency: z.string().min(1).max(3).default("USD"),
    start_date: z.string().optional(),
    end_date: z.string().optional(),
  })
  .refine(
    (data) => {
      if (data.start_date && data.end_date) {
        return new Date(data.end_date) > new Date(data.start_date);
      }
      return true;
    },
    { message: "End date must be after start date", path: ["end_date"] }
  );

type CreateCampaignFormValues = z.infer<typeof createCampaignSchema>;

interface CreateCampaignDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateCampaignDialog({
  open,
  onOpenChange,
}: CreateCampaignDialogProps) {
  const router = useRouter();
  const { mutateAsync: createCampaign } = useCreateCampaign();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<CreateCampaignFormValues>({
    resolver: zodResolver(createCampaignSchema),
    defaultValues: { currency: "USD" },
  });

  async function onSubmit(values: CreateCampaignFormValues) {
    const budgetCents =
      values.budget_cents ? Math.round(parseFloat(values.budget_cents) * 100) : null;
    try {
      const campaign = await createCampaign({
        name: values.name,
        budget_cents: budgetCents,
        currency: values.currency,
        start_date: values.start_date || null,
        end_date: values.end_date || null,
      });
      toast.success(`Campaign "${campaign.name}" created`);
      reset();
      onOpenChange(false);
      router.push(`/campaigns/${campaign.id}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create campaign");
    }
  }

  function handleOpenChange(next: boolean) {
    if (!next) reset();
    onOpenChange(next);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Create New Campaign</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="campaign-name">Campaign name</Label>
            <Input
              id="campaign-name"
              placeholder="Summer 2026"
              disabled={isSubmitting}
              {...register("name")}
            />
            {errors.name && (
              <p className="text-xs text-destructive">{errors.name.message}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="campaign-budget">Budget (optional)</Label>
              <Input
                id="campaign-budget"
                type="number"
                step="0.01"
                min="0"
                placeholder="500.00"
                disabled={isSubmitting}
                {...register("budget_cents")}
              />
              {errors.budget_cents && (
                <p className="text-xs text-destructive">
                  {errors.budget_cents.message}
                </p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="campaign-currency">Currency</Label>
              <Input
                id="campaign-currency"
                placeholder="USD"
                maxLength={3}
                disabled={isSubmitting}
                {...register("currency")}
              />
              {errors.currency && (
                <p className="text-xs text-destructive">{errors.currency.message}</p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="campaign-start">Start date (optional)</Label>
              <Input
                id="campaign-start"
                type="date"
                disabled={isSubmitting}
                {...register("start_date")}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="campaign-end">End date (optional)</Label>
              <Input
                id="campaign-end"
                type="date"
                disabled={isSubmitting}
                {...register("end_date")}
              />
              {errors.end_date && (
                <p className="text-xs text-destructive">{errors.end_date.message}</p>
              )}
            </div>
          </div>

          <DialogFooter className="gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Creating..." : "Create campaign"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
