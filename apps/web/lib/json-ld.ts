/**
 * jsonLdScript menserialisasi data JSON-LD untuk disisipkan ke <script>.
 *
 * `JSON.stringify` TIDAK meng-escape "<". Nilai yang berisi "</script>"
 * akan menutup elemen script lebih awal, dan sisanya diperlakukan browser
 * sebagai HTML — jadi payloads seperti `</script><script>...` bisa
 * menyisipkan JavaScript ke halaman.
 *
 * Untuk karya, field seperti judul & deskripsi berasal dari user, jadi
 * ini bukan teori: siapa pun yang bisa membuat karya bisa menyisipkan
 * script ke halaman publik karya itu (stored XSS).
 *
 * Next.js mendokumentasikan hal yang sama: ganti "<" dengan "\u003c"
 * sebelum masuk ke __html.
 */
export function jsonLdScript(data: unknown): string {
  return JSON.stringify(data).replace(/</g, "\\u003c");
}
