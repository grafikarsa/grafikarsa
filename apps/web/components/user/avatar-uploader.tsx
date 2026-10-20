"use client";

// Alias: "Image" di file ini dipakai sebagai constructor DOM (untuk crop),
// jadi komponen Next.js diimpor dengan nama lain.
import NextImage from "next/image";

import Cropper from "react-easy-crop";
import { useCallback, useRef, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { useUpload } from "@/components/portfolio/block-form";

// helper: potong gambar sesuai area pilihan lalu kembalikan File hasil crop.
async function cropToFile(
  src: string,
  area: { x: number; y: number; width: number; height: number },
  ratio: number,
): Promise<File> {
  const img = new Image();
  img.crossOrigin = "anonymous";
  await new Promise<void>((resolve, reject) => {
    img.onload = () => resolve();
    img.onerror = () => reject(new Error("Gagal memuat gambar."));
    img.src = src;
  });
  const canvas = document.createElement("canvas");
  // Simpan resolusi wajar (maks 1200px sisi panjang) agar tidak membuang kuota.
  const targetW = Math.min(area.width, 1200);
  const targetH = targetW / ratio;
  canvas.width = Math.round(targetW);
  canvas.height = Math.round(targetH);
  const ctx = canvas.getContext("2d");
  if (!ctx) throw new Error("Canvas tidak tersedia.");
  ctx.drawImage(img, area.x, area.y, area.width, area.height, 0, 0, canvas.width, canvas.height);
  const blob: Blob | null = await new Promise((resolve) =>
    canvas.toBlob(resolve, "image/jpeg", 0.9),
  );
  if (!blob) throw new Error("Gagal memproses gambar.");
  return new File([blob], "cropped.jpg", { type: "image/jpeg" });
}

// AvatarUploader: pilih file -> crop 1:1 -> unggah -> confirm link_to=avatar.
export function AvatarUploader({
  currentMediaID,
  onDone,
}: {
  currentMediaID?: string | null;
  onDone: (mediaID: string) => void;
}) {
  const [src, setSrc] = useState<string | null>(null);
  const [crop, setCrop] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [areaPixels, setAreaPixels] = useState<{
    x: number;
    y: number;
    width: number;
    height: number;
  } | null>(null);
  const [busy, setBusy] = useState(false);
  const { upload, progress } = useUpload();
  const fileRef = useRef<HTMLInputElement | null>(null);

  const onCropComplete = useCallback(
    (_area: unknown, pixels: { x: number; y: number; width: number; height: number }) => {
      setAreaPixels(pixels);
    },
    [],
  );

  async function confirmCrop() {
    if (!src || !areaPixels) return;
    setBusy(true);
    try {
      const file = await cropToFile(src, areaPixels, 1);
      const res = await upload("avatar", file);
      const { apiSend } = await import("@/lib/api");
      await apiSend(`v1/media/${res.media_id}/confirm`, "POST", { link_to: "avatar" });
      toast.success("Avatar diperbarui.");
      setSrc(null);
      onDone(res.media_id);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal mengunggah avatar.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        {currentMediaID && (
          <NextImage
            src={`/api/v1/media/${currentMediaID}?w=320`}
            alt="Avatar sekarang"
            width={64}
            height={64}
            className="size-16 rounded-full object-cover"
          />
        )}
        <input
          ref={fileRef}
          type="file"
          accept="image/jpeg,image/png,image/webp"
          aria-label="Pilih foto profil"
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (!f) return;
            setSrc(URL.createObjectURL(f));
          }}
        />
      </div>

      {src && (
        <div className="flex flex-col gap-2">
          <div className="relative h-64 w-full overflow-hidden rounded border">
            <Cropper
              image={src}
              crop={crop}
              zoom={zoom}
              aspect={1}
              cropShape="round"
              showGrid={false}
              onCropChange={setCrop}
              onZoomChange={setZoom}
              onCropComplete={onCropComplete}
              restrictPosition
            />
          </div>
          <label className="text-xs" htmlFor="zoom">
            Zoom
          </label>
          <input
            id="zoom"
            type="range"
            min={1}
            max={3}
            step={0.1}
            value={zoom}
            onChange={(e) => setZoom(Number(e.target.value))}
          />
          {progress !== null && <p className="text-xs">Mengunggah… {progress}%</p>}
          <div className="flex gap-2">
            <Button onClick={confirmCrop} disabled={busy}>
              Simpan avatar
            </Button>
            <Button
              variant="ghost"
              onClick={() => {
                setSrc(null);
                if (fileRef.current) fileRef.current.value = "";
              }}
              disabled={busy}
            >
              Batal
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
