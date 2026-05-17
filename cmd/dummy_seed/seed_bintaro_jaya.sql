-- Seed: Bintaro Jaya — Jaya Real Properti — Ciputat, Tangerang Selatan
-- Region Kota Tangerang Selatan id = 289

DO $$
DECLARE
    dev_jrp_id   UUID;
    loc_bjaya_id UUID;
    proj_bjaya_id UUID;
BEGIN
    -- Cleanup jika data lama ada
    DELETE FROM t_project_unit_types WHERE project_id IN (
        SELECT id FROM t_projects WHERE slug = 'bintaro-jaya'
    );
    DELETE FROM t_projects       WHERE slug = 'bintaro-jaya';
    DELETE FROM t_project_locations WHERE name = 'Bintaro Jaya';
    DELETE FROM t_users          WHERE email = 'developer@jayarealproperti.com';
    DELETE FROM t_developers     WHERE slug = 'jaya-real-properti';

    -- 1. Developer
    INSERT INTO t_developers (
        company_name, slug, email, owner_name, phone,
        logo, cover_image, address, description, website,
        package_type, status
    ) VALUES (
        'Jaya Real Properti',
        'jaya-real-properti',
        'developer@jayarealproperti.com',
        'Hendra Jaya',
        '+6221-7453900',
        'https://images.unsplash.com/photo-1560179707-f14e90ef3623?auto=format&fit=crop&q=80&w=200&h=200',
        'https://images.unsplash.com/photo-1560448204-e02f11c3d0e2?auto=format&fit=crop&q=80&w=1200',
        'Gedung Jaya, Jl. MH Thamrin No.12, Jakarta Pusat',
        'Jaya Real Properti adalah pengembang properti berpengalaman sejak 1970 yang telah membangun ribuan unit hunian berkualitas di area Bintaro dan sekitarnya. Dengan komitmen pada kualitas dan keberlanjutan, kami menghadirkan hunian impian bagi keluarga Indonesia.',
        'https://www.jayarealproperti.co.id',
        'premium',
        'active'
    ) RETURNING id INTO dev_jrp_id;

    -- 2. User Developer
    INSERT INTO t_users (name, email, password, role, developer_id)
    VALUES (
        'Jaya Real Properti',
        'developer@jayarealproperti.com',
        '$2a$10$ZIOeOgJtsmGFi0S5EwtY8ehu/r4nZV8k7TjfjEkZ1ZKohYzt1URWS',
        'developer',
        dev_jrp_id
    );

    -- 3. Lokasi — Ciputat, Tangerang Selatan (region_id 289)
    INSERT INTO t_project_locations (name, latitude, longitude, address, region_id)
    VALUES (
        'Bintaro Jaya',
        -6.2756,
        106.7228,
        'Jl. Bintaro Utama, Ciputat, Tangerang Selatan',
        289
    ) RETURNING id INTO loc_bjaya_id;

    -- 4. Proyek
    INSERT INTO t_projects (
        developer_id, project_name, slug, location_id,
        starting_price, project_type, description, cover_image,
        promo_text, featured, status, polygon_coordinates
    ) VALUES (
        dev_jrp_id,
        'Bintaro Jaya',
        'bintaro-jaya',
        loc_bjaya_id,
        980000000,
        'Residential',
        'Bintaro Jaya adalah kawasan hunian terpadu seluas lebih dari 6.000 hektare yang menghadirkan konsep kota mandiri modern di selatan Jakarta. Dilengkapi dengan infrastruktur lengkap, pusat perbelanjaan, sekolah internasional, rumah sakit, dan taman kota yang hijau, Bintaro Jaya menjadi pilihan terbaik bagi keluarga urban yang mendambakan kenyamanan dan aksesibilitas.',
        'https://images.unsplash.com/photo-1600596542815-ffad4c1539a9?auto=format&fit=crop&q=80&w=1200',
        'KPR 0% 2 Tahun Pertama',
        true,
        'active',
        '{"type":"Polygon","coordinates":[[[106.7178,-6.2820],[106.7298,-6.2820],[106.7298,-6.2680],[106.7178,-6.2680],[106.7178,-6.2820]]]}'
    ) RETURNING id INTO proj_bjaya_id;

    -- 5. Galeri
    INSERT INTO t_project_galleries (project_id, image, title, sort_order) VALUES
    (proj_bjaya_id, 'https://images.unsplash.com/photo-1600585154340-be6161a56a0c?auto=format&fit=crop&q=80&w=800', 'Eksterior Cluster', 1),
    (proj_bjaya_id, 'https://images.unsplash.com/photo-1600607687939-ce8a6c25118c?auto=format&fit=crop&q=80&w=800', 'Ruang Keluarga', 2),
    (proj_bjaya_id, 'https://images.unsplash.com/photo-1600566753086-00f18fb6b3ea?auto=format&fit=crop&q=80&w=800', 'Dapur Modern', 3),
    (proj_bjaya_id, 'https://images.unsplash.com/photo-1600607687644-c7171b42498b?auto=format&fit=crop&q=80&w=800', 'Master Bedroom', 4),
    (proj_bjaya_id, 'https://images.unsplash.com/photo-1571939228382-b2f2b585ce15?auto=format&fit=crop&q=80&w=800', 'Area Taman & Kolam', 5);

    -- 6. Tipe Unit (5 tipe)
    INSERT INTO t_project_unit_types (project_id, type_name, land_size, building_size, bedroom, bathroom, garage, price, stock, slug) VALUES
    (proj_bjaya_id, 'Anggrek 6/10', '60', '48', 2, 1, 1, 980000000,  20, 'anggrek-6x10'),
    (proj_bjaya_id, 'Bougenville 7/12', '84', '65', 3, 2, 1, 1350000000, 16, 'bougenville-7x12'),
    (proj_bjaya_id, 'Cempaka 8/15', '120', '96', 3, 2, 2, 1850000000, 12, 'cempaka-8x15'),
    (proj_bjaya_id, 'Dahlia 10/18', '180', '145', 4, 3, 2, 2800000000,  8, 'dahlia-10x18'),
    (proj_bjaya_id, 'Eidelweis Grand Villa', '260', '210', 5, 4, 3, 4200000000,  4, 'eidelweis-grand-villa');

    -- 7. Unit Spesifik (block A & B, 5 unit per tipe kecil, 3 per tipe besar)
    INSERT INTO t_project_units (project_id, unit_type_id, block, number, status, price)
    SELECT ut.project_id, ut.id,
           CASE WHEN g % 2 = 0 THEN 'B' ELSE 'A' END,
           g::text,
           (CASE WHEN g = 1 THEN 'booked' ELSE 'available' END)::unit_status,
           ut.price
    FROM t_project_unit_types ut, generate_series(1, 5) AS g
    WHERE ut.project_id = proj_bjaya_id
      AND ut.slug IN ('anggrek-6x10', 'bougenville-7x12');

    INSERT INTO t_project_units (project_id, unit_type_id, block, number, status, price)
    SELECT ut.project_id, ut.id, 'A', g::text, 'available', ut.price
    FROM t_project_unit_types ut, generate_series(1, 3) AS g
    WHERE ut.project_id = proj_bjaya_id
      AND ut.slug IN ('cempaka-8x15', 'dahlia-10x18', 'eidelweis-grand-villa');

    RAISE NOTICE 'Seed Bintaro Jaya selesai. Developer ID: %, Project ID: %', dev_jrp_id, proj_bjaya_id;
END $$;
