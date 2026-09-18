-- 邀请返利：代理商体系（一级代理/二级代理）。
-- agent_level: 0=普通用户 1=一级代理（管理员设置） 2=二级代理（一级代理设置）
-- agent_parent_user_id: 二级代理所属的一级代理 user_id（一级代理为 NULL）
-- show_full_email: 管理员控制该用户在自己的邀请返利页面是否能查看邀请用户完整邮箱
-- hide_affiliate_for_self: 该账号自身的邀请返利入口是否被隐藏（管理员或上级代理单独控制）

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS agent_level SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS agent_parent_user_id BIGINT NULL;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS show_full_email BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE user_affiliates
    ADD COLUMN IF NOT EXISTS hide_affiliate_for_self BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_agent_parent
    ON user_affiliates (agent_parent_user_id)
    WHERE agent_parent_user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_agent_level
    ON user_affiliates (agent_level)
    WHERE agent_level > 0;

CREATE INDEX IF NOT EXISTS idx_user_affiliates_admin_settings_v4
    ON user_affiliates (updated_at)
    WHERE aff_code_custom = true
       OR aff_rebate_rate_percent IS NOT NULL
       OR aff_rebate_freeze_hours IS NOT NULL
       OR aff_rebate_duration_days IS NOT NULL
       OR hide_affiliate_for_invitees = true
       OR hide_affiliate_for_self = true
       OR agent_level > 0
       OR show_full_email = true;

COMMENT ON COLUMN user_affiliates.agent_level IS '代理商等级：0=普通 1=一级代理 2=二级代理';
COMMENT ON COLUMN user_affiliates.agent_parent_user_id IS '二级代理所属一级代理的用户ID';
COMMENT ON COLUMN user_affiliates.show_full_email IS '该用户邀请返利页面是否可查看邀请用户完整邮箱';
COMMENT ON COLUMN user_affiliates.hide_affiliate_for_self IS '该账号自身的邀请返利入口是否被隐藏';
