import { useMutation, useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { startAuthentication } from "@simplewebauthn/browser";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api, setAuthToken } from "@/shared/api/client";
import { setAccessToken } from "@/shared/auth/session";

type AuthConfig = {
  github_enabled: boolean;
  gitlab_enabled: boolean;
  passkey_enabled: boolean;
  email_enabled: boolean;
  phone_enabled: boolean;
  dev_otp?: boolean;
};

type SendCodeResponse = { ok: boolean; dev_code?: string };

export function LoginPage() {
  const navigate = useNavigate();
  const [tab, setTab] = useState<"email" | "phone">("email");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [step, setStep] = useState<"input" | "code">("input");

  const { data: config } = useQuery({
    queryKey: ["auth-config"],
    queryFn: async () => (await api.get<AuthConfig>("/auth/config")).data,
  });

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const token = params.get("access_token");
    const err = params.get("error");
    if (err) {
      toast.error(err);
      window.history.replaceState({}, "", "/login");
    }
    if (token) {
      setAccessToken(token);
      setAuthToken(token);
      window.history.replaceState({}, "", "/login");
      navigate({ to: "/workspaces" });
    }
  }, [navigate]);

  const onAuthed = (accessToken: string) => {
    setAccessToken(accessToken);
    setAuthToken(accessToken);
    navigate({ to: "/workspaces" });
  };

  const sendCode = useMutation({
    mutationFn: async () => {
      const path = tab === "email" ? "/auth/email/send-code" : "/auth/phone/send-code";
      const body = tab === "email" ? { email } : { phone };
      const { data } = await api.post<SendCodeResponse>(path, body);
      return data;
    },
    onSuccess: (data) => {
      setStep("code");
      if (data.dev_code) {
        setCode(data.dev_code);
        toast.success(`Код для dev: ${data.dev_code}`, { duration: 120_000 });
      } else {
        toast.success("Код отправлен");
      }
    },
    onError: () => toast.error("Не удалось отправить код"),
  });

  const verifyCode = useMutation({
    mutationFn: async () => {
      const path = tab === "email" ? "/auth/email/verify" : "/auth/phone/verify";
      const body = tab === "email" ? { email, code } : { phone, code };
      const { data } = await api.post<{ access_token: string }>(path, body);
      return data.access_token;
    },
    onSuccess: onAuthed,
    onError: () => toast.error("Неверный код"),
  });

  const passkeyLogin = useMutation({
    mutationFn: async () => {
      const { data: begin } = await api.post<{ options: unknown; session_id: string }>(
        "/auth/passkey/login/begin",
      );
      const credential = await startAuthentication({ optionsJSON: begin.options as never });
      const { data: finish } = await api.post<{ access_token: string }>("/auth/passkey/login/finish", {
        session_id: begin.session_id,
        credential,
      });
      return finish.access_token;
    },
    onSuccess: onAuthed,
    onError: () => toast.error("Passkey не сработал"),
  });

  if (import.meta.env.VITE_AUTH_DEV_MODE === "true") {
    return (
      <div className="mx-auto max-w-md space-y-4 p-6">
        <h1 className="text-2xl font-bold">Лид-кот</h1>
        <p className="text-sm text-muted-foreground">Режим разработки — API без входа.</p>
        <Button className="w-full" onClick={() => navigate({ to: "/workspaces" })}>
          Войти как dev
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-md space-y-4 p-6">
      <h1 className="text-2xl font-bold">Войти в Lead Cat</h1>
      <p className="text-sm text-muted-foreground">Email, телефон, passkey или GitHub / GitLab</p>
      {config?.dev_otp && (
        <p className="rounded-lg border border-border bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
          Локальный режим: после «Получить код» код появится в уведомлении и подставится в поле (письма/SMS
          нет).
        </p>
      )}

      <Tabs
        value={tab}
        onValueChange={(v) => {
          setTab(v as "email" | "phone");
          setStep("input");
        }}
      >
        <TabsList className="w-full">
          <TabsTrigger value="email" className="flex-1">
            Email
          </TabsTrigger>
          <TabsTrigger value="phone" className="flex-1">
            Телефон
          </TabsTrigger>
        </TabsList>

        <TabsContent value="email" className="space-y-3 pt-2">
          {step === "input" ? (
            <>
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  placeholder="you@company.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
              <Button className="w-full" onClick={() => sendCode.mutate()} disabled={sendCode.isPending}>
                {sendCode.isPending ? <Spinner className="mr-2" /> : null}
                Получить код
              </Button>
            </>
          ) : (
            <>
              <div className="space-y-2">
                <Label htmlFor="code-email">Код</Label>
                <Input
                  id="code-email"
                  placeholder="123456"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                />
              </div>
              <Button className="w-full" onClick={() => verifyCode.mutate()} disabled={verifyCode.isPending}>
                {verifyCode.isPending ? <Spinner className="mr-2" /> : null}
                Войти
              </Button>
            </>
          )}
        </TabsContent>

        <TabsContent value="phone" className="space-y-3 pt-2">
          {step === "input" ? (
            <>
              <div className="space-y-2">
                <Label htmlFor="phone">Телефон</Label>
                <Input
                  id="phone"
                  placeholder="+77001234567"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                />
              </div>
              <Button className="w-full" onClick={() => sendCode.mutate()} disabled={sendCode.isPending}>
                {sendCode.isPending ? <Spinner className="mr-2" /> : null}
                Получить код
              </Button>
            </>
          ) : (
            <>
              <div className="space-y-2">
                <Label htmlFor="code-phone">Код</Label>
                <Input
                  id="code-phone"
                  placeholder="123456"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                />
              </div>
              <Button className="w-full" onClick={() => verifyCode.mutate()} disabled={verifyCode.isPending}>
                {verifyCode.isPending ? <Spinner className="mr-2" /> : null}
                Войти
              </Button>
            </>
          )}
        </TabsContent>
      </Tabs>

      {config?.passkey_enabled && (
        <Button
          variant="outline"
          className="w-full"
          onClick={() => passkeyLogin.mutate()}
          disabled={passkeyLogin.isPending}
        >
          {passkeyLogin.isPending ? <Spinner className="mr-2" /> : null}
          Войти с Passkey
        </Button>
      )}

      <div className="flex gap-2">
        {config?.github_enabled && (
          <Button variant="outline" className="flex-1" asChild>
            <a href="/api/auth/oauth/github">GitHub</a>
          </Button>
        )}
        {config?.gitlab_enabled && (
          <Button variant="outline" className="flex-1" asChild>
            <a href="/api/auth/oauth/gitlab">GitLab</a>
          </Button>
        )}
      </div>
    </div>
  );
}
