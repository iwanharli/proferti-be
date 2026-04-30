-- Clean up existing data
TRUNCATE t_developers, t_users, t_projects, t_unit_types, t_units, t_galleries, t_leads, t_lead_notes, t_bookings, t_transactions, t_campaigns, t_traffic_logs, t_subscriptions, t_settings, t_notifications, na_accounts, na_sessions, na_verification_tokens CASCADE;

-- Seed Developers
INSERT INTO t_developers (id, company_name, slug, owner_name, phone, email, logo, cover_image, address, city, description, website, package_type)
VALUES 
('d0000000-0000-4000-a000-000000000001', 'Agung Sedayu Group', 'agung-sedayu', 'Sugianto Kusuma', '021-23580000', 'info@agungsedayu.com', 'https://upload.wikimedia.org/wikipedia/id/thumb/7/7a/Agung_Sedayu_Group_logo.svg/1200px-Agung_Sedayu_Group_logo.svg.png', 'https://images.unsplash.com/photo-1486406146926-c627a92ad1ab?auto=format&fit=crop&q=80', 'Pantai Indah Kapuk', 'Jakarta Utara', 'Developer properti terkemuka di Indonesia.', 'https://www.agungsedayu.com', 'Premium'),
('d0000000-0000-4000-a000-000000000002', 'Sinar Mas Land', 'sinar-mas-land', 'Muktar Widjaja', '021-50368368', 'customer.care@sinarmasland.com', 'https://upload.wikimedia.org/wikipedia/id/thumb/a/a4/Sinar_Mas_Land_logo.svg/1200px-Sinar_Mas_Land_logo.svg.png', 'https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&q=80', 'BSD City', 'Tangerang', 'Membangun masa depan Indonesia.', 'https://www.sinarmasland.com', 'Premium');

-- Seed Users
INSERT INTO t_users (id, developer_id, name, email, password, role)
VALUES 
('f0000000-0000-4000-a000-000000000001', 'd0000000-0000-4000-a000-000000000001', 'Admin Sedayu', 'admin@sedayu.com', 'hashed_password', 'admin'),
('f0000000-0000-4000-a000-000000000002', 'd0000000-0000-4000-a000-000000000002', 'Marketing SML', 'marketing@sml.com', 'hashed_password', 'developer');

-- Seed Projects
INSERT INTO t_projects (id, developer_id, project_name, slug, city, district, address, latitude, longitude, description, cover_image, starting_price, promo_text, featured)
VALUES 
('e0000000-0000-4000-a000-000000000001', 'd0000000-0000-4000-a000-000000000001', 'Pik 2 Tokyo Riverside', 'pik-2-tokyo-riverside', 'Jakarta Utara', 'PIK 2', 'Jl. Marina Indah', -6.1111, 106.7777, 'Apartemen mewah dengan konsep gaya hidup Jepang.', 'https://images.unsplash.com/photo-1545324418-cc1a3fa10c00?auto=format&fit=crop&q=80', 500000000, 'Cicilan 5 Juta per Bulan', true),
('e0000000-0000-4000-a000-000000000002', 'd0000000-0000-4000-a000-000000000002', 'BSD City The Zora', 'bsd-the-zora', 'Tangerang', 'BSD', 'Jl. BSD Grand Boulevard', -6.3000, 106.6500, 'Hunian eksklusif hasil kolaborasi Sinar Mas Land dan Mitsubishi Corporation.', 'https://images.unsplash.com/photo-1512917774080-9991f1c4c750?auto=format&fit=crop&q=80', 4000000000, 'Free Smart Home System', true);

-- Seed Unit Types
INSERT INTO t_unit_types (id, project_id, type_name, land_size, building_size, bedroom, bathroom, garage, price, stock)
VALUES 
('c1000000-0000-4000-a000-000000000001', 'e0000000-0000-4000-a000-000000000001', 'Studio', '21', '21', 1, 1, 0, 500000000, 100),
('c1000000-0000-4000-a000-000000000002', 'e0000000-0000-4000-a000-000000000001', '2 Bedroom', '45', '45', 2, 1, 0, 950000000, 50),
('c1000000-0000-4000-a000-000000000003', 'e0000000-0000-4000-a000-000000000002', 'Type Kimora', '120', '148', 3, 3, 2, 4200000000, 10);

-- Seed Units
INSERT INTO t_units (id, project_id, unit_type_id, block, number, facing, price, status)
VALUES 
('b1000000-0000-4000-a000-000000000001', 'e0000000-0000-4000-a000-000000000001', 'c1000000-0000-4000-a000-000000000001', 'A', '101', 'North', 500000000, 'available'),
('b1000000-0000-4000-a000-000000000002', 'e0000000-0000-4000-a000-000000000001', 'c1000000-0000-4000-a000-000000000001', 'A', '102', 'North', 500000000, 'available'),
('b1000000-0000-4000-a000-000000000003', 'e0000000-0000-4000-a000-000000000002', 'c1000000-0000-4000-a000-000000000003', 'K', '01', 'South', 4200000000, 'available');

-- Seed Galleries
INSERT INTO t_galleries (project_id, image, title)
VALUES 
('e0000000-0000-4000-a000-000000000001', 'https://images.unsplash.com/photo-1493663284031-b7e3aefcae8e?auto=format&fit=crop&q=80', 'Interior Living Room'),
('e0000000-0000-4000-a000-000000000002', 'https://images.unsplash.com/photo-1580587771525-78b9dba3b914?auto=format&fit=crop&q=80', 'Main Facade');

-- Seed Leads
INSERT INTO t_leads (id, developer_id, project_id, unit_type_id, name, phone, email, city, budget, source, status)
VALUES 
('a1000000-0000-4000-a000-000000000001', 'd0000000-0000-4000-a000-000000000001', 'e0000000-0000-4000-a000-000000000001', 'c1000000-0000-4000-a000-000000000001', 'Budi Santoso', '08123456789', 'budi@gmail.com', 'Jakarta', 500000000, 'Facebook Ad', 'new'),
('a1000000-0000-4000-a000-000000000002', 'd0000000-0000-4000-a000-000000000002', 'e0000000-0000-4000-a000-000000000002', 'c1000000-0000-4000-a000-000000000003', 'Siska Amelia', '08119988776', 'siska@yahoo.com', 'Tangerang', 4500000000, 'Website', 'hot');

-- Seed Lead Notes
INSERT INTO t_lead_notes (lead_id, user_id, note)
VALUES 
('a1000000-0000-4000-a000-000000000001', 'f0000000-0000-4000-a000-000000000001', 'Tertarik dengan unit studio, mau visit hari sabtu.'),
('a1000000-0000-4000-a000-000000000002', 'f0000000-0000-4000-a000-000000000002', 'Sudah visit lokasi, sedang pertimbangkan opsi KPR.');

-- Seed Bookings
INSERT INTO t_bookings (id, lead_id, unit_id, booking_fee, payment_method, payment_status, booking_date)
VALUES 
('91000000-0000-4000-a000-000000000001', 'a1000000-0000-4000-a000-000000000001', 'b1000000-0000-4000-a000-000000000001', 5000000, 'Transfer Bank', 'paid', CURRENT_DATE);

-- Seed Settings
INSERT INTO t_settings ("key", value)
VALUES 
('site_name', 'Proferti Marketplace'),
('contact_email', 'support@proferti.com'),
('maintenance_mode', 'false');

-- Seed Notifications
INSERT INTO t_notifications (user_id, title, message)
VALUES 
('f0000000-0000-4000-a000-000000000001', 'Lead Baru!', 'Ada lead baru masuk dari Facebook Ad: Budi Santoso.'),
('f0000000-0000-4000-a000-000000000002', 'Booking Berhasil', 'Lead Siska Amelia telah melakukan booking unit K-01.');


