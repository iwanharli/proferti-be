-- Clean up existing data
TRUNCATE t_developers, t_users, t_projects, t_unit_types, t_galleries CASCADE;

-- Seed Developers
INSERT INTO t_developers (id, company_name, slug, owner_name, phone, email, logo, cover_image, address, city, description, website, package_type)
VALUES 
('d0000000-0000-4000-a000-000000000001', 'Agung Sedayu Group', 'agung-sedayu', 'Sugianto Kusuma', '021-23580000', 'info@agungsedayu.com', 'https://upload.wikimedia.org/wikipedia/id/thumb/7/7a/Agung_Sedayu_Group_logo.svg/1200px-Agung_Sedayu_Group_logo.svg.png', 'https://images.unsplash.com/photo-1486406146926-c627a92ad1ab?auto=format&fit=crop&q=80', 'Pantai Indah Kapuk', 'Jakarta Utara', 'Developer properti terkemuka di Indonesia.', 'https://www.agungsedayu.com', 'Premium'),
('d0000000-0000-4000-a000-000000000002', 'Sinar Mas Land', 'sinar-mas-land', 'Muktar Widjaja', '021-50368368', 'customer.care@sinarmasland.com', 'https://upload.wikimedia.org/wikipedia/id/thumb/a/a4/Sinar_Mas_Land_logo.svg/1200px-Sinar_Mas_Land_logo.svg.png', 'https://images.unsplash.com/photo-1497366216548-37526070297c?auto=format&fit=crop&q=80', 'BSD City', 'Tangerang', 'Membangun masa depan Indonesia.', 'https://www.sinarmasland.com', 'Premium');

-- Seed Users (Linked to Developers)
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
('c0000000-0000-4000-a000-000000000001', 'e0000000-0000-4000-a000-000000000001', 'Studio', '21', '21', 1, 1, 0, 500000000, 100),
('c0000000-0000-4000-a000-000000000002', 'e0000000-0000-4000-a000-000000000001', '2 Bedroom', '45', '45', 2, 1, 0, 950000000, 50),
('c0000000-0000-4000-a000-000000000003', 'e0000000-0000-4000-a000-000000000002', 'Type Kimora', '120', '148', 3, 3, 2, 4200000000, 10);

-- Seed Galleries
INSERT INTO t_galleries (project_id, image, title)
VALUES 
('e0000000-0000-4000-a000-000000000001', 'https://images.unsplash.com/photo-1493663284031-b7e3aefcae8e?auto=format&fit=crop&q=80', 'Interior Living Room'),
('e0000000-0000-4000-a000-000000000001', 'https://images.unsplash.com/photo-1502672260266-1c1ef2d93688?auto=format&fit=crop&q=80', 'Master Bedroom'),
('e0000000-0000-4000-a000-000000000002', 'https://images.unsplash.com/photo-1580587771525-78b9dba3b914?auto=format&fit=crop&q=80', 'Main Facade');

