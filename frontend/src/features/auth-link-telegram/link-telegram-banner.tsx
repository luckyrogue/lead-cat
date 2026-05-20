import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { api } from "@/shared/api/client";

declare global {
  interface Window {
    Telegram?: { WebApp?: { initData?: string } };
  }
}

export function LinkTelegramBanner() {
  const qc = useQueryClient();
  const link = useMutation({
    mutationFn: async () => {
      const initData = window.Telegram?.WebApp?.initData;
      if (!initData) throw new Error("no initData");
      await api.post("/me/link-telegram", null, {
        headers: { "X-Telegram-Init-Data": initData },
      });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["me"] });
      toast.success("Telegram привязан");
    },
    onError: () => toast.error("Не удалось привязать Telegram"),
  });

  if (!window.Telegram?.WebApp?.initData) return null;

  return (
    <Button
      variant="outline"
      className="mb-2 w-full"
      onClick={() => link.mutate()}
      disabled={link.isPending}
    >
      {link.isPending ? <Spinner className="mr-2" /> : null}
      Привязать Telegram к аккаунту
    </Button>
  );
}
