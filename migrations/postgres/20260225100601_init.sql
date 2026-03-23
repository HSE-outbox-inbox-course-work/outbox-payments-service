-- +goose Up
-- +goose StatementBegin
create table accounts
(
    id         uuid primary key not null,
    username   text             not null,
    balance    bigint           not null,
    updated_at timestamptz      not null default now(),
    created_at timestamptz      not null default now()
);

create table transfers
(
    id              uuid primary key not null,
    from_account_id uuid             not null references accounts (id),
    to_account_id   uuid             not null references accounts (id),
    amount          bigint           not null check (amount > 0),
    created_at      timestamptz      not null default now()
);

create table outbox
(
    id           uuid primary key not null,
    aggregate_id uuid             not null,
    event_type   text             not null,
    payload      jsonb            not null,
    status       text             not null check (status in ('new', 'sent')),
    created_at   timestamptz      not null default now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
drop table accounts;
drop table transfers;
drop table outbox;
-- +goose StatementEnd
