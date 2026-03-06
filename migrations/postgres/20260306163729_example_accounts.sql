-- +goose Up
insert into accounts(id, username, balance)
values ('11111111-1111-1111-1111-111111111111', 'sasha', 100),
       ('22222222-2222-2222-2222-222222222222', 'denis', 200);

-- +goose Down
delete from accounts where id in ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222')
