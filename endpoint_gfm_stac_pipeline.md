# Endpoint yang Digunakan untuk Pipeline Copernicus GFM

Dokumen ini menjelaskan endpoint yang perlu dipakai untuk mengambil data **Copernicus Global Flood Monitoring (GFM)** dari STAC API, mengolahnya menjadi polygon banjir, menyimpannya ke database, dan menampilkannya di aplikasi.

---

## 1. Ringkasan Alur

```text
Frontend / Worker
   ↓
STAC API EODC
   ↓
Cari item GFM berdasarkan AOI + tanggal
   ↓
Ambil asset ensemble_flood_extent
   ↓
Baca/download GeoTIFF
   ↓
Clip ke AOI
   ↓
Polygonize raster
   ↓
Simpan ke PostGIS
   ↓
Tampilkan sebagai history / layer peta
```

---

## 2. External Endpoint: STAC API EODC

Base URL:

```text
https://stac.eodc.eu/api/v1
```

Collection yang digunakan:

```text
GFM
```

---

## 3. Endpoint 1 — Cek Metadata Collection

Endpoint:

```http
GET https://stac.eodc.eu/api/v1/collections/GFM
```

Gunanya:

- memastikan collection `GFM` tersedia;
- melihat daftar asset/band yang tersedia;
- melihat temporal extent data;
- melihat item assets seperti `ensemble_flood_extent`, `ensemble_likelihood`, `exclusion_mask`, dan lain-lain.

Contoh:

```bash
curl -s "https://stac.eodc.eu/api/v1/collections/GFM" | jq
```

Endpoint ini **tidak dipakai terus-menerus** dalam pipeline. Cukup untuk validasi awal atau debugging.

---

## 4. Endpoint 2 — Search Item GFM Berdasarkan AOI dan Tanggal

Endpoint utama:

```http
POST https://stac.eodc.eu/api/v1/search
```

Ini adalah endpoint paling penting.

Gunanya:

- mencari item GFM berdasarkan area;
- mencari item GFM berdasarkan rentang waktu;
- mendapatkan `item.id`;
- mendapatkan daftar asset yang tersedia untuk setiap item.

Contoh request untuk Jakarta:

```bash
curl -s -X POST "https://stac.eodc.eu/api/v1/search" \
  -H "Content-Type: application/json" \
  -d '{
    "collections": ["GFM"],
    "datetime": "2026-04-01T00:00:00Z/2026-05-02T23:59:59Z",
    "limit": 5,
    "intersects": {
      "type": "Polygon",
      "coordinates": [[
        [106.60, -6.40],
        [107.10, -6.40],
        [107.10, -5.90],
        [106.60, -5.90],
        [106.60, -6.40]
      ]]
    }
  }' | jq '.features[] | {
    id: .id,
    datetime: .properties.datetime,
    assets: (.assets | keys)
  }'
```

Contoh output penting:

```json
{
  "id": "ENSEMBLE_FLOOD_20260501T110659_VV_OC020M_E042N087T3",
  "datetime": "2026-05-01T11:06:59Z",
  "assets": [
    "ensemble_flood_extent",
    "ensemble_likelihood",
    "ensemble_water_extent",
    "exclusion_mask",
    "reference_water_mask"
  ]
}
```

Dalam aplikasi, endpoint ini dipakai oleh **worker ingestion**, bukan langsung oleh frontend.

---

## 5. Endpoint 3 — Ambil Detail Item Berdasarkan ID

Endpoint:

```http
GET https://stac.eodc.eu/api/v1/collections/GFM/items/{ITEM_ID}
```

Contoh:

```bash
ITEM_ID="ENSEMBLE_FLOOD_20260501T110659_VV_OC020M_E042N087T3"

curl -s "https://stac.eodc.eu/api/v1/collections/GFM/items/${ITEM_ID}" | jq
```

Gunanya:

- mengambil detail satu item;
- mengambil URL asset GeoTIFF;
- menyimpan metadata item ke tabel `gfm_scene`;
- mengambil `properties.datetime` untuk `acquisition_time`.

---

## 6. Endpoint 4 — Ambil URL Asset GeoTIFF

Asset yang paling penting:

```text
ensemble_flood_extent
```

Cara mengambil URL-nya:

```bash
ITEM_ID="ENSEMBLE_FLOOD_20260501T110659_VV_OC020M_E042N087T3"

ASSET_URL=$(curl -s "https://stac.eodc.eu/api/v1/collections/GFM/items/${ITEM_ID}" \
  | jq -r '.assets.ensemble_flood_extent.href')

echo "$ASSET_URL"
```

Hasilnya adalah URL Cloud Optimized GeoTIFF.

Catatan penting:

```text
ASSET_URL bukan endpoint API JSON.
ASSET_URL adalah file raster/COG yang dibaca oleh GDAL.
```

---

## 7. Asset yang Disarankan

Untuk MVP:

| Asset | Fungsi |
|---|---|
| `ensemble_flood_extent` | Data utama area banjir |

Untuk versi lebih matang:

| Asset | Fungsi |
|---|---|
| `ensemble_flood_extent` | Polygon area banjir |
| `ensemble_likelihood` | Confidence / tingkat keyakinan |
| `exclusion_mask` | Mask area yang kurang valid untuk deteksi |
| `ensemble_water_extent` | Seluruh air terdeteksi, termasuk air permanen |
| `reference_water_mask` | Air normal/permanen sebagai pembanding |

Rekomendasi awal:

```text
Mulai dari ensemble_flood_extent saja.
Tambahkan ensemble_likelihood dan exclusion_mask setelah pipeline utama stabil.
```

---

## 8. Cara Baca Asset GeoTIFF dengan GDAL

Cek metadata remote COG:

```bash
gdalinfo "/vsicurl/${ASSET_URL}"
```

Clip ke bbox Jakarta:

```bash
mkdir -p data/gfm

gdalwarp \
  -overwrite \
  -t_srs EPSG:4326 \
  -te_srs EPSG:4326 \
  -te 106.60 -6.40 107.10 -5.90 \
  -r near \
  -of GTiff \
  "/vsicurl/${ASSET_URL}" \
  data/gfm/jakarta_ensemble_flood_extent.tif
```

Cek min/max value:

```bash
gdalinfo -mm data/gfm/jakarta_ensemble_flood_extent.tif
```

Jika min/max hanya `0`, kemungkinan tidak ada flood pixel di AOI/tanggal itu.

---

## 9. Polygonize Raster

```bash
gdal_polygonize.py \
  data/gfm/jakarta_ensemble_flood_extent.tif \
  -f GeoJSON \
  data/gfm/jakarta_flood_extent.geojson \
  flood_polygons \
  flood_value
```

Hasilnya GeoJSON dengan attribute:

```text
flood_value
```

Biasanya yang dipakai untuk area banjir:

```sql
WHERE flood_value = 1
```

Tetapi tetap cek dulu nilai yang muncul.

---

## 10. Import ke PostGIS Staging

```bash
ogr2ogr \
  -f PostgreSQL \
  PG:"host=localhost port=5432 dbname=flood user=postgres password=postgres" \
  data/gfm/jakarta_flood_extent.geojson \
  -nln gfm_flood_polygon_stage \
  -overwrite
```

Cek value:

```sql
SELECT flood_value, COUNT(*)
FROM gfm_flood_polygon_stage
GROUP BY flood_value
ORDER BY flood_value;
```

---

## 11. Insert ke Tabel Final

```sql
INSERT INTO gfm_flood_polygon (
    scene_id,
    acquisition_time,
    geom,
    centroid,
    area_m2
)
SELECT
    $1::uuid,
    $2::timestamptz,
    ST_Multi(ST_MakeValid(geom)),
    ST_Centroid(ST_MakeValid(geom)),
    ST_Area(ST_Transform(ST_MakeValid(geom), 3857))
FROM gfm_flood_polygon_stage
WHERE flood_value = 1
  AND ST_Area(ST_Transform(ST_MakeValid(geom), 3857)) >= 500;
```

---

## 12. Internal Endpoint yang Perlu Dibuat di Aplikasi

Selain external STAC API, aplikasi kamu sebaiknya punya endpoint internal sendiri.

### 12.1 Search Scene GFM

```http
GET /api/gfm/scenes?from=2026-04-01&to=2026-05-02&bbox=106.60,-6.40,107.10,-5.90
```

Gunanya:

- menampilkan daftar scene GFM yang sudah tersimpan di database;
- debugging ingestion;
- melihat tanggal akuisisi yang tersedia.

Response contoh:

```json
[
  {
    "id": "uuid",
    "stac_item_id": "ENSEMBLE_FLOOD_20260501T110659_VV_OC020M_E042N087T3",
    "acquisition_time": "2026-05-01T11:06:59Z",
    "product_time": "2026-05-01T12:00:00Z"
  }
]
```

---

### 12.2 Trigger Ingestion Manual

```http
POST /api/gfm/ingest
```

Body:

```json
{
  "from": "2026-04-01T00:00:00Z",
  "to": "2026-05-02T23:59:59Z",
  "bbox": [106.60, -6.40, 107.10, -5.90]
}
```

Gunanya:

- menjalankan ingestion manual;
- testing sebelum scheduler aktif;
- mengambil item baru dari STAC API.

Endpoint ini memanggil:

```text
POST https://stac.eodc.eu/api/v1/search
```

lalu memproses item yang belum ada di `gfm_scene`.

---

### 12.3 Ambil Polygon Banjir untuk Peta

```http
GET /api/flood-polygons?from=2026-05-01T00:00:00Z&to=2026-05-02T00:00:00Z&bbox=106.60,-6.40,107.10,-5.90
```

Gunanya:

- menampilkan polygon banjir di frontend;
- menjadi source layer MapLibre/Leaflet;
- mendukung time slider.

Response contoh:

```json
{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "geometry": {
        "type": "MultiPolygon",
        "coordinates": []
      },
      "properties": {
        "id": "uuid",
        "source": "copernicus_gfm",
        "acquisition_time": "2026-05-01T11:06:59Z",
        "area_m2": 12345.67,
        "admin_city_code": "31.71",
        "admin_district_code": "31.71.01",
        "admin_village_code": "31.71.01.1001"
      }
    }
  ]
}
```

---

### 12.4 Ambil Summary per Wilayah

```http
GET /api/flood-summary/admin/{admin_code}?from=2026-04-01&to=2026-05-02
```

Contoh:

```http
GET /api/flood-summary/admin/31.71?from=2026-04-01&to=2026-05-02
```

Gunanya:

- menampilkan history banjir per kota/kecamatan/kelurahan;
- chart luas banjir per hari;
- ranking wilayah terdampak.

Response contoh:

```json
[
  {
    "date": "2026-05-01",
    "flood_polygon_count": 3,
    "total_flood_area_m2": 42000.5,
    "max_flood_area_m2": 20000.0,
    "first_detected_at": "2026-05-01T11:06:59Z",
    "last_detected_at": "2026-05-01T11:06:59Z"
  }
]
```

---

### 12.5 Ambil Region Boundary

```http
GET /api/regions?level=city
GET /api/regions?level=district&parent_code=31.71
GET /api/regions?level=village&parent_code=31.71.01
```

Gunanya:

- dropdown wilayah;
- filter peta;
- lookup nama wilayah berdasarkan code.

---

## 13. Endpoint yang Tidak Perlu Dipakai Dulu

### WMS-T

WMS-T cocok kalau ingin langsung menampilkan layer raster GFM sebagai overlay.

Namun untuk pipeline history kamu, WMS-T bukan prioritas karena:

- sulit dipakai untuk analitik;
- tidak langsung menghasilkan polygon;
- tidak ideal untuk summary per wilayah.

Gunakan STAC + GeoTIFF terlebih dahulu.

### GFM REST API v2

GFM juga punya REST API khusus, tetapi untuk workflow kamu, STAC lebih sederhana karena:

- langsung bisa search by AOI dan waktu;
- langsung memberi item dan asset;
- cocok untuk ingestion ke database.

---

## 14. Urutan Endpoint yang Dipakai dalam Worker

```text
1. POST /search
   External: https://stac.eodc.eu/api/v1/search
   Tujuan: cari item GFM terbaru untuk AOI + tanggal.

2. GET /collections/GFM/items/{ITEM_ID}
   External: https://stac.eodc.eu/api/v1/collections/GFM/items/{ITEM_ID}
   Tujuan: ambil detail item dan asset href.

3. GET asset href
   External: URL dari assets.ensemble_flood_extent.href
   Tujuan: baca/download GeoTIFF.

4. Internal DB insert
   Simpan ke gfm_scene, gfm_asset, gfm_flood_polygon.

5. GET /api/flood-polygons
   Internal aplikasi
   Tujuan: frontend mengambil polygon banjir.

6. GET /api/flood-summary/admin/{admin_code}
   Internal aplikasi
   Tujuan: frontend mengambil history per wilayah.
```

---

## 15. Rekomendasi Final

Untuk MVP, gunakan endpoint berikut saja:

### External

```text
POST https://stac.eodc.eu/api/v1/search
GET  https://stac.eodc.eu/api/v1/collections/GFM/items/{ITEM_ID}
GET  {assets.ensemble_flood_extent.href}
```

### Internal

```text
POST /api/gfm/ingest
GET  /api/gfm/scenes
GET  /api/flood-polygons
GET  /api/flood-summary/admin/{admin_code}
GET  /api/regions
```

Jangan mulai dari WMS-T. Untuk kebutuhan history dan database, mulai dari **STAC search → GeoTIFF asset → polygonize → PostGIS**.

