DROP TRIGGER IF EXISTS apps_updated_at ON apps;
DROP TRIGGER IF EXISTS users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at();

DROP INDEX IF EXISTS idx_oauth_states_expires_at;
DROP INDEX IF EXISTS idx_api_tokens_user_id;
DROP INDEX IF EXISTS idx_domains_app_id;
DROP INDEX IF EXISTS idx_deployments_created_at;
DROP INDEX IF EXISTS idx_deployments_app_id;
DROP INDEX IF EXISTS idx_apps_user_id;

DROP TABLE IF EXISTS oauth_states;
DROP TABLE IF EXISTS domains;
DROP TABLE IF EXISTS deployments;
DROP TABLE IF EXISTS apps;
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS users;
