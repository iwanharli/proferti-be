-- SEEDER DATA DEMO PROFERTI
-- Menghapus data lama (Opsional, pastikan tabel users dikecualikan jika ingin mempertahankan user lain)
-- TRUNCATE t_project_unit_types, t_projects, t_project_locations, t_developers, t_users CASCADE;

DO $$ 
DECLARE 
    dev_sm_id UUID;
    dev_pm_id UUID;
    proj_sm_id UUID;
    proj_pm_id UUID;
    loc_sm_id UUID;
    loc_pm_id UUID;
BEGIN 
    -- Cleanup old demo data to avoid conflicts (optional but recommended for seeders)
    -- We truncate with CASCADE to clean up dependent tables
    DELETE FROM t_project_unit_types WHERE project_id IN (SELECT id FROM t_projects WHERE slug IN ('sinarmas-land-plaza', 'paramount-gading-serpong'));
    DELETE FROM t_projects WHERE slug IN ('sinarmas-land-plaza', 'paramount-gading-serpong');
    DELETE FROM t_project_locations WHERE name IN ('BSD City', 'Gading Serpong');
    DELETE FROM t_users WHERE email IN ('admindemo@gmail.com', 'developersinarmasdemo@gmail.com', 'developerparamountdemo@gmail.com');
    DELETE FROM t_developers WHERE slug IN ('sinarmas', 'paramount', 'sinarmas-land-2026', 'paramount-land-2026');

    -- 1. Buat Developer
    INSERT INTO t_developers (company_name, slug, email, status) 
    VALUES ('Sinar Mas Land', 'sinarmas-land-2026', 'sinarmas@demo.com', 'active') 
    RETURNING id INTO dev_sm_id;

    INSERT INTO t_developers (company_name, slug, email, status) 
    VALUES ('Paramount Land', 'paramount-land-2026', 'paramount@demo.com', 'active') 
    RETURNING id INTO dev_pm_id;

    -- 2. Buat User Demo (Password sudah ter-hash BCrypt)
    -- Admin (Password: admindemo)
    INSERT INTO t_users (name, email, password, role) 
    VALUES ('Admin Demo', 'admindemo@gmail.com', '$2a$10$5OnYXoWgjYhv3YHse6baf.Y/zW8rKxKcyQdyre.oF.25cyWYgfqFe', 'admin');
    
    -- Dev Sinar Mas (Password: sinarmasdemo)
    INSERT INTO t_users (name, email, password, role, developer_id) 
    VALUES ('Sinar Mas Developer', 'developersinarmasdemo@gmail.com', '$2a$10$ZIOeOgJtsmGFi0S5EwtY8ehu/r4nZV8k7TjfjEkZ1ZKohYzt1URWS', 'developer', dev_sm_id);
    
    -- Dev Paramount (Password: paramountdemo)
    INSERT INTO t_users (name, email, password, role, developer_id) 
    VALUES ('Paramount Developer', 'developerparamountdemo@gmail.com', '$2a$10$WhkoofA6Ol0zOstnvtorkelsxInkoXRlYI1OeCw2GxvLvIgzZdkWa', 'developer', dev_pm_id);

    -- 3. Buat Lokasi Proyek (Kabupaten Tangerang ID: 284)
    INSERT INTO t_project_locations (name, latitude, longitude, address, region_id) 
    VALUES ('BSD City', -6.2960364, 106.6391285, 'BSD City, Tangerang', 284) 
    RETURNING id INTO loc_sm_id;

    INSERT INTO t_project_locations (name, latitude, longitude, address, region_id) 
    VALUES ('Gading Serpong', -6.2727769, 106.6198221, 'Gading Serpong, Tangerang', 284) 
    RETURNING id INTO loc_pm_id;

    -- 4. Buat Proyek
    INSERT INTO t_projects (developer_id, project_name, slug, location_id, starting_price, project_type, description, cover_image, promo_text, featured, status, polygon_coordinates) 
    VALUES (
        dev_sm_id, 
        'Sinar Mas Land Plaza BSD City', 
        'sinarmas-land-plaza', 
        loc_sm_id, 
        1500000000, 
        'Commercial/Office',
        'Pusat perkantoran modern di jantung BSD City dengan fasilitas lengkap dan akses strategis.',
        'https://images.unsplash.com/photo-1486406146926-c627a92ad1ab?auto=format&fit=crop&q=80&w=1000',
        'Sewa 2 Tahun Gratis 1 Bulan',
        true,
        'active',
        '{"type":"Polygon","coordinates":[[[106.6381, -6.2970], [106.6401, -6.2970], [106.6401, -6.2950], [106.6381, -6.2950], [106.6381, -6.2970]]]}'
    );

    INSERT INTO t_projects (developer_id, project_name, slug, location_id, starting_price, project_type, description, cover_image, promo_text, featured, status, polygon_coordinates) 
    VALUES (
        dev_pm_id, 
        'Paramount Gading Serpong', 
        'paramount-gading-serpong', 
        loc_pm_id, 
        1200000000, 
        'Residential',
        'Hunian mewah dengan konsep green living dan fasilitas komunitas eksklusif di Gading Serpong.',
        'https://images.unsplash.com/photo-1512917774080-9991f1c4c750?auto=format&fit=crop&q=80&w=1000',
        'DP 0% & Free Biaya KPR',
        true,
        'active',
        '{"type":"Polygon","coordinates":[[[106.6188, -6.2737], [106.6208, -6.2737], [106.6208, -6.2717], [106.6188, -6.2717], [106.6188, -6.2737]]]}'
    );
    
    -- Get project IDs back for galleries and units
    SELECT id INTO proj_sm_id FROM t_projects WHERE slug = 'sinarmas-land-plaza';
    SELECT id INTO proj_pm_id FROM t_projects WHERE slug = 'paramount-gading-serpong';

    -- 4a. Buat Galeri Proyek
    INSERT INTO t_project_galleries (project_id, image, title, sort_order) VALUES
    (proj_sm_id, 'https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&q=80&w=800', 'Lobby Utama', 1),
    (proj_sm_id, 'https://images.unsplash.com/photo-1497215728101-856f4ea42174?auto=format&fit=crop&q=80&w=800', 'Ruang Kantor', 2),
    (proj_pm_id, 'https://images.unsplash.com/photo-1613490493576-7fde63acd811?auto=format&fit=crop&q=80&w=800', 'Fasilitas Kolam Renang', 1),
    (proj_pm_id, 'https://images.unsplash.com/photo-1613977257363-707ba934f246?auto=format&fit=crop&q=80&w=800', 'Interior Ruang Tamu', 2);

    -- 5. Buat Tipe Unit (5 per proyek)
    INSERT INTO t_project_unit_types (project_id, type_name, land_size, building_size, bedroom, bathroom, garage, price, stock, slug) VALUES 
    (proj_sm_id, 'Type A - Deluxe', '90', '72', 3, 2, 1, 1500000000, 10, 'type-a-deluxe'),
    (proj_sm_id, 'Type B - Premium', '120', '90', 4, 3, 2, 1850000000, 8, 'type-b-premium'),
    (proj_sm_id, 'Type C - Executive', '150', '120', 4, 4, 2, 2500000000, 5, 'type-c-executive'),
    (proj_sm_id, 'Type D - Suite', '200', '180', 5, 5, 3, 3800000000, 3, 'type-d-suite'),
    (proj_sm_id, 'Type E - Penthouse', '300', '250', 5, 6, 4, 5500000000, 2, 'type-e-penthouse');

    INSERT INTO t_project_unit_types (project_id, type_name, land_size, building_size, bedroom, bathroom, garage, price, stock, slug) VALUES 
    (proj_pm_id, 'Standard - 6x10', '60', '45', 2, 1, 1, 1200000000, 15, 'standard-6x10'),
    (proj_pm_id, 'Superior - 7x12', '84', '60', 3, 2, 1, 1600000000, 12, 'superior-7x12'),
    (proj_pm_id, 'Luxury - 8x15', '120', '95', 3, 3, 2, 2100000000, 10, 'luxury-8x15'),
    (proj_pm_id, 'Grand - 10x18', '180', '140', 4, 4, 2, 3200000000, 6, 'grand-10x18'),
    (proj_pm_id, 'Elite Villa', '250', '200', 5, 5, 3, 4500000000, 4, 'elite-villa');

    -- 6. Buat Unit Spesifik
    INSERT INTO t_project_units (project_id, unit_type_id, block, number, status, price)
    SELECT project_id, id, 'A', (ROW_NUMBER() OVER())::text, 'available', price
    FROM t_project_unit_types WHERE slug = 'type-a-deluxe' LIMIT 5;

    INSERT INTO t_project_units (project_id, unit_type_id, block, number, status, price)
    SELECT project_id, id, 'STD', (ROW_NUMBER() OVER())::text, 'available', price
    FROM t_project_unit_types WHERE slug = 'standard-6x10' LIMIT 5;

END $$;
