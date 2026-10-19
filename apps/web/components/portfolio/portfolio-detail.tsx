"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { apiGet, apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

type Block = { id: string; block_type: string; position: number; payload: Record<string, unknown> };
type Portfolio = {
  id: string;
  title: string;
  slug: string;
  status: string;
  blocks: Block[];
  like_count: number;
  comment_count: number;
  is_liked: boolean;
};

type Comment = {
  id: string;
  body_md: string;
  username: string;
  author_name: string;
  children: Comment[];
};

function CommentNode({
  c,
  onReply,
}: {
  c: Comment;
  onReply: (parent: string, text: string) => void;
}) {
  const [text, setText] = useState("");
  const [open, setOpen] = useState(false);
  return (
    <div className="border-l pl-3">
      <p className="text-sm font-semibold">
        {c.author_name} <span className="text-muted-foreground font-normal">@{c.username}</span>
      </p>
      <p className="text-sm">{c.body_md}</p>
      <Button variant="ghost" size="sm" onClick={() => setOpen(!open)}>
        Balas
      </Button>
      {open && (
        <div className="flex gap-2">
          <Input
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Tulis balasan..."
          />
          <Button
            size="sm"
            onClick={() => {
              onReply(c.id, text);
              setText("");
              setOpen(false);
            }}
          >
            Kirim
          </Button>
        </div>
      )}
      {c.children.map((ch) => (
        <CommentNode key={ch.id} c={ch} onReply={onReply} />
      ))}
    </div>
  );
}

export function PortfolioDetail({ initial }: { initial: Portfolio }) {
  const user = useAuthStore((s) => s.user);
  const [p, setP] = useState(initial);
  const [comments, setComments] = useState<Comment[]>([]);
  const [text, setText] = useState("");

  async function reload() {
    const data = await apiGet<Portfolio>(`v1/portfolios/id/${p.id}`);
    setP(data);
    const c = await apiGet<Comment[]>(`v1/portfolios/id/${p.id}/comments`);
    setComments(c);
  }

  async function like() {
    try {
      await apiSend(
        `v1/portfolios/id/${p.id}/${p.is_liked ? "like" : "like"}`,
        p.is_liked ? "DELETE" : "POST",
      );
      reload();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  async function comment(parent?: string, body?: string) {
    const content = body ?? text;
    if (!content.trim()) return;
    try {
      await apiSend(`v1/portfolios/id/${p.id}/comments`, "POST", { parent_id: parent, content });
      setText("");
      reload();
      toast.success("Komentar terkirim.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Gagal.");
    }
  }

  return (
    <main className="mx-auto flex max-w-2xl flex-col gap-4 p-4">
      <h1 className="text-3xl font-bold">{p.title}</h1>
      <p className="text-muted-foreground text-sm">
        {p.like_count} suka · {p.comment_count} komentar · {p.status}
      </p>
      {p.blocks.map((b) => (
        <BlockView key={b.id} b={b} />
      ))}
      {user && (
        <Button variant="outline" onClick={like}>
          {p.is_liked ? "Batal suka" : "Suka"}
        </Button>
      )}
      <Card>
        <CardContent className="flex flex-col gap-3 p-4">
          <h2 className="font-semibold">Komentar</h2>
          {user && (
            <div className="flex gap-2">
              <Input
                value={text}
                onChange={(e) => setText(e.target.value)}
                placeholder="Tulis komentar..."
              />
              <Button onClick={() => comment()}>Kirim</Button>
            </div>
          )}
          {comments.map((c) => (
            <CommentNode key={c.id} c={c} onReply={(parent, t) => comment(parent, t)} />
          ))}
        </CardContent>
      </Card>
    </main>
  );
}

function BlockView({ b }: { b: Block }) {
  const p = b.payload as Record<string, string>;
  const table = b.payload as unknown as { headers?: string[]; rows?: string[][] };
  switch (b.block_type) {
    case "text":
      return (
        <div className="prose prose-sm dark:prose-invert max-w-none">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{p.markdown ?? ""}</ReactMarkdown>
        </div>
      );
    case "image": {
      const src = p.media_id ? `/api/v1/media/${p.media_id}?w=800` : (p.url ?? "");
      if (!src) return null;
      return (
        <figure className="flex flex-col gap-1">
          <img src={src} alt={p.alt ?? ""} className="rounded" loading="lazy" />
          {p.caption && (
            <figcaption className="text-muted-foreground text-xs">{p.caption}</figcaption>
          )}
        </figure>
      );
    }
    case "table":
      return (
        <div className="overflow-auto">
          <Table>
            <TableHeader>
              <TableRow>
                {(table.headers ?? []).map((h, i) => (
                  <TableHead key={i}>{h}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {(table.rows ?? []).map((r, i) => (
                <TableRow key={i}>
                  {(r ?? []).map((c, j) => (
                    <TableCell key={j}>{c}</TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      );
    case "youtube":
      return (
        <iframe
          src={`https://www.youtube-nocookie.com/embed/${p.video_id}`}
          className="aspect-video w-full rounded"
          loading="lazy"
          sandbox="allow-scripts allow-same-origin"
          title="YouTube"
        />
      );
    case "button":
      return (
        <a
          href={p.url}
          target="_blank"
          rel="noreferrer"
          className={buttonVariants({
            variant: p.variant === "secondary" ? "secondary" : "default",
          })}
        >
          {p.label ?? "Buka"}
        </a>
      );
    case "link":
      return (
        <a href={p.url} target="_blank" rel="noreferrer" className="text-sm underline">
          {p.title || p.url}
        </a>
      );
    case "pdf": {
      // Preview halaman-1 dibuat worker; kalau belum ada, tampilkan tautan file.
      const previewSrc = p.media_id ? `/api/v1/media/${p.media_id}?preview=1` : "";
      const fileHref = p.media_id ? `/api/v1/media/${p.media_id}` : (p.url ?? "");
      if (!fileHref) return null;
      return (
        <div className="flex flex-col gap-2">
          {previewSrc && (
            <a href={fileHref} target="_blank" rel="noreferrer">
              <img
                src={previewSrc}
                alt="Pratinjau dokumen PDF"
                className="max-h-96 rounded border object-contain"
                loading="lazy"
              />
            </a>
          )}
          <a href={fileHref} target="_blank" rel="noreferrer" className="text-sm underline">
            Buka dokumen PDF
          </a>
        </div>
      );
    }
    case "ppt":
    case "doc":
    case "canva":
    case "figma":
    case "embed":
      if (!p.url) return null;
      return (
        <iframe
          src={p.url}
          className="aspect-video w-full rounded"
          loading="lazy"
          sandbox="allow-scripts allow-same-origin"
          referrerPolicy="no-referrer"
          title={b.block_type}
        />
      );
    default:
      return null;
  }
}
