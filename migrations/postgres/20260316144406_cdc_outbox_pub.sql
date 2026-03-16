-- +goose Up
create publication outbox_pub for table outbox;

-- +goose Down
SELECT 'down SQL query';
