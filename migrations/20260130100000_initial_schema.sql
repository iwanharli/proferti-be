-- +goose Up
-- gen_random_uuid() tersedia di PostgreSQL 13+ tanpa extension.

CREATE TYPE user_role AS ENUM ('admin', 'developer');
CREATE TYPE project_status AS ENUM ('active', 'soldout', 'upcoming');
CREATE TYPE unit_status AS ENUM ('available', 'booked', 'sold');

CREATE TABLE t_developers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    owner_name VARCHAR(255),
    phone VARCHAR(50),
    email VARCHAR(255),
    logo TEXT,
    cover_image TEXT,
    address TEXT,
    city VARCHAR(120),
    description TEXT,
    website VARCHAR(512),
    package_type VARCHAR(64),
    expired_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE t_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    developer_id UUID REFERENCES t_developers (id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(50),
    password VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'developer',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_developer_id ON t_users (developer_id);
CREATE INDEX idx_users_email ON t_users (email);

CREATE TABLE t_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    developer_id UUID NOT NULL REFERENCES t_developers (id) ON DELETE CASCADE,
    project_name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    city VARCHAR(120),
    district VARCHAR(120),
    address TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    description TEXT,
    cover_image TEXT,
    starting_price NUMERIC(18, 2) NOT NULL DEFAULT 0,
    promo_text VARCHAR(512),
    status project_status NOT NULL DEFAULT 'active',
    featured BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (developer_id, slug)
);

CREATE INDEX idx_projects_developer_id ON t_projects (developer_id);
CREATE INDEX idx_projects_status ON t_projects (status);
CREATE INDEX idx_projects_featured ON t_projects (featured) WHERE featured = true;

CREATE TABLE t_unit_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES t_projects (id) ON DELETE CASCADE,
    type_name VARCHAR(120) NOT NULL,
    land_size VARCHAR(64),
    building_size VARCHAR(64),
    bedroom SMALLINT,
    bathroom SMALLINT,
    garage SMALLINT,
    price NUMERIC(18, 2) NOT NULL DEFAULT 0,
    stock INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_unit_types_project_id ON t_unit_types (project_id);

CREATE TABLE t_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES t_projects (id) ON DELETE CASCADE,
    unit_type_id UUID NOT NULL REFERENCES t_unit_types (id) ON DELETE RESTRICT,
    block VARCHAR(32),
    number VARCHAR(32),
    facing VARCHAR(64),
    price NUMERIC(18, 2) NOT NULL DEFAULT 0,
    status unit_status NOT NULL DEFAULT 'available',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_units_project_id ON t_units (project_id);
CREATE INDEX idx_units_unit_type_id ON t_units (unit_type_id);
CREATE INDEX idx_units_status ON t_units (status);

CREATE TABLE t_galleries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES t_projects (id) ON DELETE CASCADE,
    image TEXT NOT NULL,
    title VARCHAR(255),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_galleries_project_id ON t_galleries (project_id);

CREATE TABLE t_leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    developer_id UUID NOT NULL REFERENCES t_developers (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES t_projects (id) ON DELETE CASCADE,
    unit_type_id UUID REFERENCES t_unit_types (id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    email VARCHAR(255),
    city VARCHAR(120),
    budget NUMERIC(18, 2),
    message TEXT,
    source VARCHAR(64),
    status VARCHAR(32) NOT NULL DEFAULT 'new',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_leads_developer_id ON t_leads (developer_id);
CREATE INDEX idx_leads_project_id ON t_leads (project_id);
CREATE INDEX idx_leads_status ON t_leads (status);

CREATE TABLE t_lead_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id UUID NOT NULL REFERENCES t_leads (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES t_users (id) ON DELETE CASCADE,
    note TEXT NOT NULL,
    next_followup_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_lead_notes_lead_id ON t_lead_notes (lead_id);
CREATE INDEX idx_lead_notes_user_id ON t_lead_notes (user_id);

CREATE TABLE t_bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id UUID NOT NULL REFERENCES t_leads (id) ON DELETE CASCADE,
    unit_id UUID NOT NULL REFERENCES t_units (id) ON DELETE RESTRICT,
    booking_fee NUMERIC(18, 2) NOT NULL DEFAULT 0,
    payment_method VARCHAR(64),
    payment_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    booking_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_bookings_lead_id ON t_bookings (lead_id);
CREATE INDEX idx_bookings_unit_id ON t_bookings (unit_id);

CREATE TABLE t_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES t_bookings (id) ON DELETE CASCADE,
    final_price NUMERIC(18, 2) NOT NULL DEFAULT 0,
    dp NUMERIC(18, 2),
    loan_method VARCHAR(64),
    bank_name VARCHAR(120),
    tenor_year SMALLINT,
    monthly_installment NUMERIC(18, 2),
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transactions_booking_id ON t_transactions (booking_id);

CREATE TABLE t_campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    developer_id UUID NOT NULL REFERENCES t_developers (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES t_projects (id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    platform VARCHAR(64),
    budget NUMERIC(18, 2),
    start_date DATE,
    end_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_campaigns_developer_id ON t_campaigns (developer_id);
CREATE INDEX idx_campaigns_project_id ON t_campaigns (project_id);

CREATE TABLE t_traffic_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES t_projects (id) ON DELETE CASCADE,
    ip_address INET,
    device VARCHAR(64),
    source VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_traffic_logs_project_id ON t_traffic_logs (project_id);
CREATE INDEX idx_traffic_logs_created_at ON t_traffic_logs (created_at);

CREATE TABLE t_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    developer_id UUID NOT NULL REFERENCES t_developers (id) ON DELETE CASCADE,
    package_name VARCHAR(120) NOT NULL,
    price NUMERIC(18, 2) NOT NULL DEFAULT 0,
    start_date DATE,
    end_date DATE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriptions_developer_id ON t_subscriptions (developer_id);

CREATE TABLE t_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "key" VARCHAR(191) NOT NULL UNIQUE,
    value TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE t_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES t_users (id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    is_read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_id ON t_notifications (user_id);
CREATE INDEX idx_notifications_is_read ON t_notifications (user_id, is_read);

-- +goose Down
DROP TABLE IF EXISTS t_notifications;
DROP TABLE IF EXISTS t_settings;
DROP TABLE IF EXISTS t_subscriptions;
DROP TABLE IF EXISTS t_traffic_logs;
DROP TABLE IF EXISTS t_campaigns;
DROP TABLE IF EXISTS t_transactions;
DROP TABLE IF EXISTS t_bookings;
DROP TABLE IF EXISTS t_lead_notes;
DROP TABLE IF EXISTS t_leads;
DROP TABLE IF EXISTS t_galleries;
DROP TABLE IF EXISTS t_units;
DROP TABLE IF EXISTS t_unit_types;
DROP TABLE IF EXISTS t_projects;
DROP TABLE IF EXISTS t_users;
DROP TABLE IF EXISTS t_developers;

DROP TYPE IF EXISTS unit_status;
DROP TYPE IF EXISTS project_status;
DROP TYPE IF EXISTS user_role;
