-- +goose Up
-- +goose StatementBegin
create table if not exists accounts
(
    id         uuid primary key not null,
    username   text             not null,
    balance    bigint           not null,
    updated_at timestamptz      not null default now(),
    created_at timestamptz      not null default now()
);

create table if not exists transfers
(
    id              uuid primary key not null,
    from_account_id uuid             not null references accounts (id),
    to_account_id   uuid             not null references accounts (id),
    amount          bigint           not null check (amount > 0),
    created_at      timestamptz      not null default now()
);

create table if not exists outbox
(
    id         uuid primary key not null,
    event_type text             not null,
    payload    jsonb            not null,
    status     text             not null check (status in ('new', 'done')),
    created_at timestamptz      not null default now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table if exists accounts;
drop table if exists transfers;
drop table if exists outbox;
-- +goose StatementEnd
