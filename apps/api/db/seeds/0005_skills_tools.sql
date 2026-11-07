-- Seed 0005: katalog skill (skills) dan tools (tools).
-- Idempoten: di-upsert berdasarkan slug.
BEGIN;

INSERT INTO skills (slug, name, category) VALUES
  ('desain-grafis',           'Desain Grafis',           'design'),
  ('tipografi',               'Tipografi',               'design'),
  ('layout',                  'Layout',                  'design'),
  ('branding',                'Branding',                'design'),
  ('ilustrasi-digital',       'Ilustrasi Digital',       'design'),
  ('ui-design',               'UI Design',               'design'),
  ('ux-research',             'UX Research',             'design'),
  ('desain-kemasan',          'Desain Kemasan',          'design'),

  ('pracetak',                'Pracetak',                'printing'),
  ('separasi-warna',          'Separasi Warna',          'printing'),
  ('cetak-offset',            'Cetak Offset',            'printing'),
  ('cetak-digital',           'Cetak Digital',           'printing'),
  ('sablon',                  'Sablon',                  'printing'),
  ('finishing-cetak',         'Finishing Cetak',         'printing'),
  ('manajemen-warna',         'Manajemen Warna',         'printing'),

  ('fotografi',               'Fotografi',               'media'),
  ('videografi',              'Videografi',              'media'),
  ('editing-video',           'Editing Video',           'media'),
  ('animasi-2d',              'Animasi 2D',              'media'),
  ('animasi-3d',              'Animasi 3D',              'media'),
  ('modeling-3d',             'Modeling 3D',             'media'),
  ('rigging',                 'Rigging',                 'media'),
  ('motion-graphic',          'Motion Graphic',          'media'),
  ('storyboard',              'Storyboard',              'media'),
  ('sound-design',            'Sound Design',            'media'),
  ('vfx',                     'VFX',                     'media'),

  ('frontend',                'Frontend',                'software'),
  ('backend',                 'Backend',                 'software'),
  ('mobile-development',      'Mobile Development',      'software'),
  ('game-development',        'Game Development',        'software'),
  ('basis-data',              'Basis Data',              'software'),
  ('api-design',              'API Design',              'software'),
  ('devops',                  'DevOps',                  'software'),
  ('testing',                 'Testing',                 'software'),

  ('administrasi-jaringan',   'Administrasi Jaringan',   'network'),
  ('administrasi-server',     'Administrasi Server',     'network'),
  ('keamanan-jaringan',       'Keamanan Jaringan',       'network'),
  ('cloud-computing',         'Cloud Computing',         'network'),
  ('troubleshooting-hardware', 'Troubleshooting Hardware', 'network'),

  ('plc',                     'PLC',                     'engineering'),
  ('pneumatik-hidrolik',      'Pneumatik & Hidrolik',    'engineering'),
  ('mikrokontroler',          'Mikrokontroler',          'engineering'),
  ('elektronika',             'Elektronika',             'engineering'),
  ('cad-mekanik',             'CAD Mekanik',             'engineering'),
  ('robotik',                 'Robotik',                 'engineering'),

  ('manajemen-gudang',        'Manajemen Gudang',        'logistics'),
  ('inventory-control',       'Inventory Control',       'logistics'),
  ('distribusi',              'Distribusi',              'logistics'),
  ('administrasi-logistik',   'Administrasi Logistik',   'logistics'),

  ('front-office',            'Front Office',            'hospitality'),
  ('housekeeping',            'Housekeeping',            'hospitality'),
  ('food-beverage-service',   'Food & Beverage Service', 'hospitality'),
  ('barista',                 'Barista',                 'hospitality'),
  ('pelayanan-tamu',          'Pelayanan Tamu',          'hospitality'),

  ('public-speaking',         'Public Speaking',         'soft_skill'),
  ('kepemimpinan',            'Kepemimpinan',            'soft_skill'),
  ('kerja-tim',               'Kerja Tim',               'soft_skill'),
  ('manajemen-proyek',        'Manajemen Proyek',        'soft_skill'),
  ('bahasa-inggris',          'Bahasa Inggris',          'soft_skill'),
  ('copywriting',             'Copywriting',             'soft_skill')
ON CONFLICT (slug) DO UPDATE
SET name     = EXCLUDED.name,
    category = EXCLUDED.category,
    is_active = true;

INSERT INTO tools (slug, name, category) VALUES
  ('adobe-photoshop',         'Adobe Photoshop',         'design_software'),
  ('adobe-illustrator',       'Adobe Illustrator',       'design_software'),
  ('adobe-indesign',          'Adobe InDesign',          'design_software'),
  ('coreldraw',               'CorelDRAW',               'design_software'),
  ('figma',                   'Figma',                   'design_software'),
  ('canva',                   'Canva',                   'design_software'),
  ('affinity-designer',       'Affinity Designer',       'design_software'),
  ('procreate',               'Procreate',               'design_software'),
  ('krita',                   'Krita',                   'design_software'),

  ('adobe-premiere-pro',      'Adobe Premiere Pro',      'media_software'),
  ('adobe-after-effects',     'Adobe After Effects',     'media_software'),
  ('davinci-resolve',         'DaVinci Resolve',         'media_software'),
  ('capcut',                  'CapCut',                  'media_software'),
  ('blender',                 'Blender',                 'media_software'),
  ('autodesk-maya',           'Autodesk Maya',           'media_software'),
  ('toon-boom-harmony',       'Toon Boom Harmony',       'media_software'),
  ('clip-studio-paint',       'Clip Studio Paint',       'media_software'),
  ('audacity',                'Audacity',                'media_software'),

  ('html',                    'HTML',                    'language'),
  ('css',                     'CSS',                     'language'),
  ('javascript',              'JavaScript',              'language'),
  ('typescript',              'TypeScript',              'language'),
  ('php',                     'PHP',                     'language'),
  ('python',                  'Python',                  'language'),
  ('go',                      'Go',                      'language'),
  ('java',                    'Java',                    'language'),
  ('kotlin',                  'Kotlin',                  'language'),
  ('dart',                    'Dart',                    'language'),
  ('cpp',                     'C++',                     'language'),
  ('csharp',                  'C#',                      'language'),
  ('sql',                     'SQL',                     'language'),

  ('react',                   'React',                   'framework'),
  ('nextjs',                  'Next.js',                 'framework'),
  ('vue',                     'Vue',                     'framework'),
  ('laravel',                 'Laravel',                 'framework'),
  ('flutter',                 'Flutter',                 'framework'),
  ('express',                 'Express',                 'framework'),
  ('tailwind-css',            'Tailwind CSS',            'framework'),
  ('unity',                   'Unity',                   'framework'),
  ('godot',                   'Godot',                   'framework'),

  ('linux',                   'Linux',                   'platform'),
  ('windows-server',          'Windows Server',          'platform'),
  ('mikrotik',                'MikroTik',                'platform'),
  ('cisco-packet-tracer',     'Cisco Packet Tracer',     'platform'),
  ('docker',                  'Docker',                  'platform'),
  ('git',                     'Git',                     'platform'),
  ('arduino-ide',             'Arduino IDE',             'platform'),

  ('arduino',                 'Arduino',                 'hardware'),
  ('esp32',                   'ESP32',                   'hardware'),
  ('raspberry-pi',            'Raspberry Pi',            'hardware'),
  ('router-switch',           'Router/Switch',           'hardware'),

  ('mesin-offset',            'Mesin Offset',            'machine'),
  ('printer-digital',         'Printer Digital',         'machine'),
  ('mesin-potong',            'Mesin Potong',            'machine'),
  ('plc-siemens',             'PLC Siemens',             'machine'),
  ('plc-omron',               'PLC Omron',               'machine')
ON CONFLICT (slug) DO UPDATE
SET name     = EXCLUDED.name,
    category = EXCLUDED.category,
    is_active = true;

COMMIT;
