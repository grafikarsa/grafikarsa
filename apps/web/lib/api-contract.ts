import type { components, paths } from "@/lib/api-types";

// Tipe turunan dari kontrak OpenAPI yang di-generate (packages/contracts/
// openapi.yaml). Pakai ini di kode FE agar perubahan backend terdeteksi saat
// typecheck, bukan saat runtime.
export type Contract = paths;
export type Schemas = components["schemas"];

export type ApiEnvelope<T> = Schemas["SuccessEnvelope"] & { data: T };
export type ApiError = Schemas["ErrorEnvelope"];
export type ErrorCode = Schemas["ErrorCode"];

export type UserContract = Schemas["User"];
export type PortfolioContract = Schemas["Portfolio"];
export type PortfolioBlockContract = Schemas["PortfolioBlock"];
export type MediaContract = Schemas["Media"];
export type LeaderboardEntryContract = Schemas["LeaderboardEntry"];
export type FeedbackThreadContract = Schemas["FeedbackThread"];
export type ChangelogContract = Schemas["Changelog"];
export type NotificationContract = Schemas["Notification"];

// MediaPurpose/BlockType dipakai editor agar tidak ada string liar.
export type MediaPurpose = NonNullable<MediaContract["purpose"]>;
export type BlockTypeContract = NonNullable<PortfolioBlockContract["block_type"]>;
