-- 邀请返利：代理可隐藏其邀请用户的邀请返利入口。
-- NULL/false 表示不限制；true 表示该用户作为邀请人时，其邀请来的用户不能再看到/使用邀请返利功能。

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS hide_affiliate_for_invitees BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_inviter_hidden_affiliate
    ON user_affiliates (inviter_id)
    WHERE inviter_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_admin_settings_v3
    ON user_affiliates (updated_at)
    WHERE aff_code_custom = true
       OR aff_rebate_rate_percent IS NOT NULL
       OR aff_rebate_freeze_hours IS NOT NULL
       OR aff_rebate_duration_days IS NOT NULL
       OR hide_affiliate_for_invitees = true;

COMMENT ON COLUMN user_affiliates.hide_affiliate_for_invitees IS '是否对该用户邀请来的账号隐藏邀请返利功能';
