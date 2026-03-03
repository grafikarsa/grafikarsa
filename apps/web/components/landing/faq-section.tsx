'use client';

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import Link from 'next/link';

const faqs = [
  {
    id: 'item-1',
    question: 'Apa itu Grafikarsa?',
    answer:
      'Grafikarsa adalah platform katalog portofolio dan social network khusus untuk warga SMKN 4 Malang. Di sini, siswa dan alumni dapat menampilkan karya mereka dan terhubung dengan komunitas.',
  },
  {
    id: 'item-2',
    question: 'Siapa yang bisa menggunakan Grafikarsa?',
    answer:
      'Grafikarsa dapat digunakan oleh siswa aktif dan alumni SMKN 4 Malang. Akun dibuat oleh admin sekolah, jadi tidak ada registrasi mandiri.',
  },
  {
    id: 'item-3',
    question: 'Bagaimana cara membuat portofolio?',
    answer:
      'Setelah login, kamu bisa membuat portofolio baru dengan menambahkan judul, thumbnail, dan berbagai blok konten seperti teks, gambar, video YouTube, tabel, dan lainnya.',
  },
  {
    id: 'item-4',
    question: 'Apakah portofolio langsung tampil di publik?',
    answer:
      'Tidak. Setelah kamu submit portofolio, admin akan mereview terlebih dahulu. Jika disetujui, portofolio akan tampil di publik.',
  },
  {
    id: 'item-5',
    question: 'Bagaimana cara mendapatkan akun?',
    answer:
      'Akun dibuat oleh admin sekolah. Hubungi guru atau admin yang bertanggung jawab untuk mendapatkan kredensial login.',
  },
];

export function FaqSection() {
  return (
    <section className="py-16 md:py-24">
      <div className="mx-auto max-w-5xl px-4 md:px-6">
        <div className="mx-auto max-w-xl text-center">
          <h2 className="text-balance text-3xl font-bold md:text-4xl lg:text-5xl">
            FAQs
          </h2>
          <p className="text-muted-foreground mt-4 text-balance">
            Temukan jawaban untuk pertanyaan yang sering diajukan tentang Grafikarsa.
          </p>
        </div>

        <div className="mx-auto mt-12 max-w-xl">
          <Accordion
            type="single"
            collapsible
            className="bg-card ring-muted w-full rounded-2xl border px-8 py-3 shadow-sm ring-4 dark:ring-0"
          >
            {faqs.map((item) => (
              <AccordionItem key={item.id} value={item.id} className="border-dashed">
                <AccordionTrigger className="cursor-pointer text-base hover:no-underline">
                  {item.question}
                </AccordionTrigger>
                <AccordionContent>
                  <p className="text-base">{item.answer}</p>
                </AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>

        </div>
      </div>
    </section>
  );
}
