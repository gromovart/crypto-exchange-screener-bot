-- persistence/postgres/migrations/006_add_subscription_foreign_key.sql
ALTER TABLE subscription_activities
ADD CONSTRAINT fk_subscription_activities_subscription_id
FOREIGN KEY (subscription_id)
REFERENCES user_subscriptions(id)
ON DELETE SET NULL;