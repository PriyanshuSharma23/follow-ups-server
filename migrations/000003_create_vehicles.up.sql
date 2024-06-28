CREATE TABLE IF NOT EXISTS vehicles (
    id bigserial PRIMARY KEY,
    license_plate varchar(20) UNIQUE,
    make varchar(50) NOT NULL,
    model varchar(50) NOT NULL,
    year int CHECK (year > 1885),
    vin varchar(17) NOT NULL UNIQUE,
    color varchar(30),
    body_type varchar(30),
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version integer NOT NULL DEFAULT 1
);