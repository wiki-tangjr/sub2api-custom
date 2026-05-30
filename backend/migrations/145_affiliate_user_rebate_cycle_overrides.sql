-- 邀请返利：专属用户返利周期覆盖配置。
-- NULL 表示沿用全局设置；非 NULL 表示该用户作为邀请人时使用自己的冻结期/有效期。

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS aff_rebate_freeze_hours INTEGER NULL;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS aff_rebate_duration_days INTEGER NULL;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_admin_settings_v2
    ON user_affiliates (updated_at)
    WHERE aff_code_custom = true
       OR aff_rebate_rate_percent IS NOT NULL
       OR aff_rebate_freeze_hours IS NOT NULL
       OR aff_rebate_duration_days IS NOT NULL;

COMMENT ON COLUMN user_affiliates.aff_rebate_freeze_hours IS '专属返利冻结期（小时，NULL 表示沿用全局）';
COMMENT ON COLUMN user_affiliates.aff_rebate_duration_days IS '专属返利有效期（天，NULL 表示沿用全局，0 表示永久）';
