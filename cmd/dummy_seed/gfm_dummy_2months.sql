-- SEEDER KHUSUS GFM (DATA 2 BULAN TERAKHIR)
-- Script ini menghasilkan data simulasi deteksi banjir untuk pengujian dashboard

DO $$ 
DECLARE 
    scene_id UUID;
    t_date DATE;
    city_record RECORD;
BEGIN 
    -- 1. Bersihkan data GFM lama (opsional)
    -- DELETE FROM gfm_admin_risk_score;
    -- DELETE FROM gfm_admin_daily_summary;
    -- DELETE FROM gfm_flood_polygon;
    -- DELETE FROM gfm_asset;
    -- DELETE FROM gfm_scene;

    -- 2. Loop untuk 60 hari terakhir
    FOR t_date IN SELECT i::date FROM generate_series(CURRENT_DATE - INTERVAL '60 days', CURRENT_DATE, '1 day') i LOOP
        
        -- Buat 1-2 scene per hari
        FOR i IN 1..2 LOOP
            INSERT INTO gfm_scene (stac_item_id, acquisition_time, product_time, platform, bbox, footprint)
            VALUES (
                'S1_GFM_' || t_date || '_' || i,
                t_date + (i * INTERVAL '10 hours'),
                t_date + (i * INTERVAL '12 hours'),
                'SENTINEL-1' || CASE WHEN i = 1 THEN 'A' ELSE 'B' END,
                ST_MakeEnvelope(106.5, -6.5, 107.5, -5.5, 4326),
                ST_SetSRID(ST_GeomFromText('MULTIPOLYGON(((106.5 -6.5, 107.5 -6.5, 107.5 -5.5, 106.5 -5.5, 106.5 -6.5)))'), 4326)
            ) RETURNING id INTO scene_id;

            -- Buat Asset dummy
            INSERT INTO gfm_asset (scene_id, band_name, asset_href)
            VALUES (scene_id, 'ensemble_flood_extent', 'https://example.com/flood_' || scene_id || '.tif');

            -- Buat Poligon Banjir acak di wilayah Tangerang/Bekasi/Bogor (Region 36.03, 32.16, 32.01)
            -- Kita ambil koordinat acak di sekitar proyek demo
            FOR j IN 1..5 LOOP
                INSERT INTO gfm_flood_polygon (
                    scene_id, acquisition_time, geom, area_m2, confidence_mean, 
                    admin_province_code, admin_city_code
                )
                VALUES (
                    scene_id,
                    t_date + (i * INTERVAL '10 hours'),
                    -- Titik acak dengan radius kecil (simulasi banjir)
                    ST_Buffer(
                        ST_SetSRID(ST_Point(106.6 + (random() * 0.4), -6.2 - (random() * 0.2)), 4326)::geography, 
                        (random() * 1000 + 500) -- Luas 500m - 1500m radius
                    )::geometry,
                    random() * 500000 + 100000, -- 10ha - 60ha
                    random() * 20 + 70, -- 70-90% confidence
                    CASE WHEN random() > 0.5 THEN '36' ELSE '32' END,
                    CASE 
                        WHEN random() < 0.3 THEN '36.03' -- Kab Tangerang
                        WHEN random() < 0.6 THEN '32.16' -- Kab Bekasi
                        ELSE '32.01' -- Kab Bogor
                    END
                );
            END LOOP;
        END LOOP;
    END LOOP;

    -- 3. Trigger Agregasi untuk mengisi gfm_admin_daily_summary
    -- Biasanya ini dilakukan di worker, tapi kita lakukan di sini untuk kemudahan seed
    INSERT INTO gfm_admin_daily_summary (
        admin_level, admin_code, admin_name, date, 
        flood_polygon_count, total_flood_area_m2, max_flood_area_m2, 
        flood_percentage, last_detected_at
    )
    SELECT 
        'city' as admin_level,
        admin_city_code as admin_code,
        CASE 
            WHEN admin_city_code = '36.03' THEN 'Kabupaten Tangerang'
            WHEN admin_city_code = '32.16' THEN 'Kabupaten Bekasi'
            WHEN admin_city_code = '32.01' THEN 'Kabupaten Bogor'
            ELSE 'Wilayah Lain'
        END as admin_name,
        detected_date as date,
        COUNT(*) as flood_polygon_count,
        SUM(area_m2) as total_flood_area_m2,
        MAX(area_m2) as max_flood_area_m2,
        (SUM(area_m2) / 10000000.0) as flood_percentage, -- Dummy percentage
        MAX(acquisition_time) as last_detected_at
    FROM gfm_flood_polygon
    GROUP BY admin_city_code, detected_date
    ON CONFLICT (source, admin_level, admin_code, date) DO UPDATE SET
        flood_polygon_count = EXCLUDED.flood_polygon_count,
        total_flood_area_m2 = EXCLUDED.total_flood_area_m2,
        flood_percentage = EXCLUDED.flood_percentage,
        last_detected_at = EXCLUDED.last_detected_at;

    -- 4. Update Risk Scores
    INSERT INTO gfm_admin_risk_score (admin_level, admin_code, admin_name, total_detections, flood_occurrence_count, risk_score, last_updated_at)
    SELECT 
        admin_level,
        admin_code,
        MAX(admin_name),
        SUM(flood_polygon_count),
        COUNT(*) FILTER (WHERE total_flood_area_m2 > 0),
        AVG(flood_percentage),
        NOW()
    FROM gfm_admin_daily_summary
    GROUP BY admin_level, admin_code
    ON CONFLICT (admin_level, admin_code) DO UPDATE SET
        total_detections = EXCLUDED.total_detections,
        risk_score = EXCLUDED.risk_score,
        last_updated_at = NOW();

END $$;
