"use client";

import { CircleAlert } from "lucide-react";

import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";
import { Button } from "@/components/ui/button";

export function QueryError({
  title = "Gagal memuat data",
  message = "Terjadi kesalahan saat memuat data. Coba lagi.",
  onRetry,
  isRetrying = false,
}: {
  title?: string;
  message?: string;
  onRetry: () => void;
  isRetrying?: boolean;
}) {
  return (
    <Alert variant="destructive">
      <CircleAlert aria-hidden="true" />
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>
        <p>{message}</p>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="mt-2"
          onClick={onRetry}
          disabled={isRetrying}
        >
          {isRetrying ? "Memuat…" : "Coba lagi"}
        </Button>
      </AlertDescription>
    </Alert>
  );
}
