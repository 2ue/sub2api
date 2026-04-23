ALTER TABLE groups
ADD COLUMN IF NOT EXISTS allow_image_generation BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN groups.allow_image_generation IS '是否允许此分组使用 OpenAI 图片生成/编辑能力';
