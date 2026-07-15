ALTER TABLE warga ADD COLUMN IF NOT EXISTS kriteria_values JSONB DEFAULT '{}';
ALTER TABLE bobot_kriteria ADD COLUMN IF NOT EXISTS bobot_values JSONB DEFAULT '{}';

-- Populate existing values
UPDATE warga SET kriteria_values = jsonb_build_object(
    'C1', COALESCE(c1_value, 0),
    'C2', COALESCE(c2_value, 0),
    'C3', COALESCE(c3_value, 0),
    'C4', COALESCE(c4_value, 0),
    'C5', COALESCE(c5_value, 0),
    'C6', COALESCE(c6_value, 0),
    'C7', COALESCE(c7_value, 0),
    'C8', COALESCE(c8_value, 0),
    'C9', COALESCE(c9_value, 0),
    'C10', COALESCE(c10_value, 0),
    'C11', COALESCE(c11_value, 0),
    'C12', COALESCE(c12_value, 0),
    'C13', COALESCE(c13_value, 0)
) WHERE kriteria_values = '{}' OR kriteria_values IS NULL;

UPDATE bobot_kriteria SET bobot_values = jsonb_build_object(
    'C1', COALESCE(bobot_c1, 0),
    'C2', COALESCE(bobot_c2, 0),
    'C3', COALESCE(bobot_c3, 0),
    'C4', COALESCE(bobot_c4, 0),
    'C5', COALESCE(bobot_c5, 0),
    'C6', COALESCE(bobot_c6, 0),
    'C7', COALESCE(bobot_c7, 0),
    'C8', COALESCE(bobot_c8, 0),
    'C9', COALESCE(bobot_c9, 0),
    'C10', COALESCE(bobot_c10, 0),
    'C11', COALESCE(bobot_c11, 0),
    'C12', COALESCE(bobot_c12, 0),
    'C13', COALESCE(bobot_c13, 0)
) WHERE bobot_values = '{}' OR bobot_values IS NULL;
