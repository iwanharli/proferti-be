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
    -- 1. Buat Developer
    INSERT INTO t_developers (company_name, slug, email, status) 
    VALUES ('Sinar Mas Land', 'sinarmas', 'sinarmas@demo.com', 'active') 
    RETURNING id INTO dev_sm_id;

    INSERT INTO t_developers (company_name, slug, email, status) 
    VALUES ('Paramount Land', 'paramount', 'paramount@demo.com', 'active') 
    RETURNING id INTO dev_pm_id;

    -- 2. Buat User Demo (Password sudah ter-hash BCrypt)
    -- Admin (Password: admindemo)
    INSERT INTO t_users (name, email, password, role) 
    VALUES ('Admin Demo', 'admindemo@gmail.com', '$2a$10$5OnYXoWgjYhv3YHse6baf.Y/zW8rKxKcyQdyre.oF.25cyWYgfqFe', 'admin');
    
    -- Dev Sinar Mas (Password: sinarmasdemo)
    INSERT INTO t_users (name, email, password, role, developer_id) 
    VALUES ('Sinar Mas Developer', 'developersinarmasdemo@gmail.com', '$2a$10$ZIOeOgJtsmGFi0S5EwtY8ehu/r4nZV8k7TjfjEkZ1ZKohYzt1URWS', 'developer', dev_sm_id);
    
    -- Dev Paramount (Password: paramount)
    INSERT INTO t_users (name, email, password, role, developer_id) 
    VALUES ('Paramount Developer', 'developerparamountdemo@gmail.com', '$2a$10$9envDibBnbMXbnufzQyZ1eypariu394ZPJ44vhdmVzeHOiYMlZl0y', 'developer', dev_pm_id);

    -- 3. Buat Lokasi Proyek
    INSERT INTO t_project_locations (name, latitude, longitude, address) 
    VALUES ('BSD City', -6.2960364, 106.6391285, 'BSD City, Tangerang') 
    RETURNING id INTO loc_sm_id;

    INSERT INTO t_project_locations (name, latitude, longitude, address) 
    VALUES ('Gading Serpong', -6.2727769, 106.6198221, 'Gading Serpong, Tangerang') 
    RETURNING id INTO loc_pm_id;

    -- 4. Buat Proyek
    INSERT INTO t_projects (developer_id, project_name, slug, location_id, starting_price, project_type) 
    VALUES (dev_sm_id, 'Sinar Mas Land Plaza BSD City', 'sinarmas-land-plaza', loc_sm_id, 1500000000, 'Commercial/Office');

    INSERT INTO t_projects (developer_id, project_name, slug, location_id, starting_price, project_type) 
    VALUES (dev_pm_id, 'Paramount Gading Serpong', 'paramount-gading-serpong', loc_pm_id, 1200000000, 'Residential');
    
    -- Get project IDs back for unit types
    SELECT id INTO proj_sm_id FROM t_projects WHERE slug = 'sinarmas-land-plaza';
    SELECT id INTO proj_pm_id FROM t_projects WHERE slug = 'paramount-gading-serpong';

    -- 5. Buat Tipe Unit (5 per proyek)
    INSERT INTO t_project_unit_types (project_id, type_name, land_size, building_size, bedroom, bathroom, price, slug) VALUES 
    (proj_sm_id, 'Type A - Deluxe', '90', '72', 3, 2, 1500000000, 'type-a-deluxe'),
    (proj_sm_id, 'Type B - Premium', '120', '90', 4, 3, 1850000000, 'type-b-premium'),
    (proj_sm_id, 'Type C - Executive', '150', '120', 4, 4, 2500000000, 'type-c-executive'),
    (proj_sm_id, 'Type D - Suite', '200', '180', 5, 5, 3800000000, 'type-d-suite'),
    (proj_sm_id, 'Type E - Penthouse', '300', '250', 5, 6, 5500000000, 'type-e-penthouse');

    INSERT INTO t_project_unit_types (project_id, type_name, land_size, building_size, bedroom, bathroom, price, slug) VALUES 
    (proj_pm_id, 'Standard - 6x10', '60', '45', 2, 1, 1200000000, 'standard-6x10'),
    (proj_pm_id, 'Superior - 7x12', '84', '60', 3, 2, 1600000000, 'superior-7x12'),
    (proj_pm_id, 'Luxury - 8x15', '120', '95', 3, 3, 2100000000, 'luxury-8x15'),
    (proj_pm_id, 'Grand - 10x18', '180', '140', 4, 4, 3200000000, 'grand-10x18'),
    (proj_pm_id, 'Elite Villa', '250', '200', 5, 5, 4500000000, 'elite-villa');

END $$;
