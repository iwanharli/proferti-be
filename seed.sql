-- TRUNCATE existing data
TRUNCATE t_project_locations, t_developers, t_users, t_projects, t_project_unit_types, t_project_units, t_project_galleries, t_leads, t_lead_notes, t_bookings, t_transactions, t_campaigns, t_traffic_logs, t_subscriptions, t_settings, t_notifications CASCADE;

-- Seed Developers (3)
INSERT INTO t_developers (id, company_name, slug, owner_name, phone, email, logo, description, website, created_at, updated_at, status)
VALUES 
('d0000000-0000-4000-a001-000000000001', 'Paramount Land', 'paramount-land', 'Ervan Adi Nugroho', '021-54200888', 'info@paramount-land.com', 'https://upload.wikimedia.org/wikipedia/commons/a/a4/Paramount_Land_Logo.png', 'Membangun hunian berkualitas di pusat Gading Serpong.', 'https://paramount-land.com', now(), now(), 'active'),
('d0000000-0000-4000-a001-000000000002', 'Ciputra Group', 'ciputra-group', 'Candra Ciputra', '021-29885858', 'marketing@ciputra.com', 'https://logo.clearbit.com/ciputra.com', 'Pelopor pengembangan properti berskala kota di Indonesia.', 'https://ciputra.com', now(), now(), 'active'),
('d0000000-0000-4000-a001-000000000003', 'Pakuwon Group', 'pakuwon-group', 'Alexander Tedja', '021-29888888', 'info@pakuwon.com', 'https://www.pakuwonjati.com/uploads/pakuwon_logo.png', 'Raja superblock dan pusat perbelanjaan di Indonesia.', 'https://pakuwon.com', now(), now(), 'active')
ON CONFLICT (id) DO NOTHING;

-- Seed Users (3)
INSERT INTO t_users (id, name, email, role, created_at, updated_at, password, developer_id)
VALUES 
('f0000000-0000-4000-a001-000000000001', 'Paramount Admin', 'admin@paramount.com', 'developer', now(), now(), '$2a$10$rN/I4T.S9.D/YjVqJq9E4.O0j6VjZ4pL6yJp6Y7q9J8qPqQpY8qP.', 'd0000000-0000-4000-a001-000000000001'),
('f0000000-0000-4000-a001-000000000002', 'Ciputra Marketing', 'marketing@ciputra.com', 'developer', now(), now(), '$2a$10$rN/I4T.S9.D/YjVqJq9E4.O0j6VjZ4pL6yJp6Y7q9J8qPqQpY8qP.', 'd0000000-0000-4000-a001-000000000002'),
('f0000000-0000-4000-a001-000000000003', 'Pakuwon Sales', 'sales@pakuwon.com', 'developer', now(), now(), '$2a$10$rN/I4T.S9.D/YjVqJq9E4.O0j6VjZ4pL6yJp6Y7q9J8qPqQpY8qP.', 'd0000000-0000-4000-a001-000000000003')
ON CONFLICT (id) DO NOTHING;

-- Dynamic Seeding for all Provinces (3 projects per province, 5 unit types per project)
DO $$
DECLARE
    prov_record RECORD;
    p_id UUID;
    l_id UUID;
    u_id UUID;
    dev_ids UUID[] := ARRAY[
        'd0000000-0000-4000-a001-000000000001'::UUID, 
        'd0000000-0000-4000-a001-000000000002'::UUID, 
        'd0000000-0000-4000-a001-000000000003'::UUID
    ];
    dev_id UUID;
    proj_names TEXT[] := ARRAY['Grand', 'Royal', 'Elite', 'Crystal', 'Emerald', 'Sapphire', 'Golden', 'Silver', 'Platinum'];
    proj_suffixes TEXT[] := ARRAY['Residences', 'Garden', 'Estate', 'Palace', 'Mansion', 'District', 'Heights', 'View'];
    unit_names TEXT[] := ARRAY['Type A', 'Type B', 'Type C', 'Type D', 'Type E'];
    i INT;
    j INT;
    p_name TEXT;
    p_slug TEXT;
    u_name TEXT;
BEGIN
    FOR prov_record IN SELECT id, name, lat, lng FROM regions WHERE LENGTH(kode) = 2 LOOP
        FOR i IN 1..3 LOOP
            p_id := gen_random_uuid();
            l_id := gen_random_uuid();
            dev_id := dev_ids[(floor(random() * 3) + 1)::int];
            p_name := proj_names[(floor(random() * 9) + 1)::int] || ' ' || proj_suffixes[(floor(random() * 8) + 1)::int] || ' ' || prov_record.name;
            IF i > 1 THEN p_name := p_name || ' ' || i; END IF;
            p_slug := lower(replace(p_name, ' ', '-')) || '-' || floor(random()*1000)::text;

            -- Insert Location (Correct Columns: name, address, latitude, longitude, region_id)
            INSERT INTO t_project_locations (id, name, address, latitude, longitude, region_id)
            VALUES (l_id, p_name || ' Site', 'Jalan Utama ' || prov_record.name, prov_record.lat + (random()-0.5)*0.15, prov_record.lng + (random()-0.5)*0.15, prov_record.id);

            -- Insert Project (Correct Columns: id, developer_id, project_name, slug, location_id, description, cover_image, starting_price, status, project_type, polygon_coordinates)
            INSERT INTO t_projects (id, developer_id, project_name, slug, location_id, description, cover_image, starting_price, status, project_type, polygon_coordinates)
            VALUES (p_id, dev_id, p_name, p_slug, l_id, 'Luxury project development in ' || prov_record.name, 'https://picsum.photos/seed/' || p_id || '/1200/800', 500000000 + (random()*2000000000), 'active', 'rumah', 
                jsonb_build_array(
                    jsonb_build_object('lat', prov_record.lat + 0.001, 'lng', prov_record.lng + 0.001),
                    jsonb_build_object('lat', prov_record.lat + 0.003, 'lng', prov_record.lng + 0.001),
                    jsonb_build_object('lat', prov_record.lat + 0.003, 'lng', prov_record.lng + 0.003),
                    jsonb_build_object('lat', prov_record.lat + 0.001, 'lng', prov_record.lng + 0.003)
                )
            );

            -- Insert 5 units types per project
            FOR j IN 1..5 LOOP
                u_id := gen_random_uuid();
                u_name := unit_names[j] || ' ' || i;
                INSERT INTO t_project_unit_types (id, project_id, type_name, slug, land_size, building_size, bedroom, bathroom, garage, price, stock)
                VALUES (u_id, p_id, u_name, p_slug || '-' || lower(replace(u_name, ' ', '-')), (60 + floor(random()*100))::text, (36 + floor(random()*80))::text, (1 + floor(random()*3))::int, (1 + floor(random()*2))::int, (1 + floor(random()*2))::int, 300000000 + (random()*1000000000), (5 + floor(random()*20))::int);
            END LOOP;
        END LOOP;
    END LOOP;
END $$;

-- Seed Settings
INSERT INTO t_settings ("key", value)
VALUES 
('site_name', 'Proferti Elite Marketplace'),
('contact_email', 'hello@proferti.com'),
('maintenance_mode', 'false')
ON CONFLICT ("key") DO UPDATE SET value = EXCLUDED.value;
