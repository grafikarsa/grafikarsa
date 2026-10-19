"use client";

import { useEffect, useRef, useState } from "react";

declare global {
  interface Window {
    turnstile?: {
      render: (el: HTMLElement, opts: Record<string, unknown>) => string;
      remove: (id: string) => void;
    };
  }
}

// Widget Turnstile (Cloudflare). Hanya dimuat bila site key dikonfigurasi,
// jadi dev tanpa Turnstile tetap jalan. Token dikirim sebagai captcha_token.
export function Turnstile({ onToken }: { onToken: (token: string) => void }) {
  const siteKey = process.env.NEXT_PUBLIC_TURNSTILE_SITE_KEY;
  const ref = useRef<HTMLDivElement | null>(null);
  const widgetId = useRef<string | null>(null);
  const [scriptReady, setScriptReady] = useState(false);

  useEffect(() => {
    if (!siteKey) return;
    if (window.turnstile) {
      setScriptReady(true);
      return;
    }
    const existing = document.querySelector<HTMLScriptElement>("script[data-turnstile]");
    if (existing) {
      existing.addEventListener("load", () => setScriptReady(true));
      return;
    }
    const s = document.createElement("script");
    s.src = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
    s.async = true;
    s.defer = true;
    s.dataset.turnstile = "1";
    s.onload = () => setScriptReady(true);
    document.head.appendChild(s);
  }, [siteKey]);

  useEffect(() => {
    if (!siteKey || !scriptReady || !ref.current || !window.turnstile) return;
    if (widgetId.current) return;
    widgetId.current = window.turnstile.render(ref.current, {
      sitekey: siteKey,
      callback: (token: string) => onToken(token),
      "expired-callback": () => onToken(""),
      "error-callback": () => onToken(""),
    });
    return () => {
      if (widgetId.current && window.turnstile) {
        window.turnstile.remove(widgetId.current);
        widgetId.current = null;
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [siteKey, scriptReady]);

  if (!siteKey) return null;
  return <div ref={ref} className="my-2" />;
}
