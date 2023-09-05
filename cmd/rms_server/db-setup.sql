create or replace database rmsgo
    character set = 'utf8mb4'
    collate = 'utf8mb4_unicode_ci';

use rmsgo;

create table users (
    id int auto_increment primary key,
    name varchar(255) not null,
    email varchar(255) unique not null,
    created_at timestamp default current_timestamp,
    deleted_at timestamp null,
    pwhash binary(255) not null -- @todo: figure out size
);

-- https://www.brightball.com/articles/automatically-reversing-account-takeovers
create table user_email_changes (
    user_id int references users(id),
    email_from varchar(255) not null,
    email_to varchar(255) not null,
    confirm_key binary(255) null, -- @todo: figure out size
    reversal_key binary(255) null, -- @todo: figure out size

    created_at timestamp default current_timestamp,
    created_ip int(11) unsigned not null,
    confirmed_at timestamp null,
    confirmed_ip int(11) unsigned null,
    reversed_at timestamp null,
    reversed_ip int(11) unsigned null
);

create table logins (
    user_id int references users(id),
    token binary(255) unique null, -- @todo: figure out size
    refresh_token binary(255) not null, -- @todo: figure out size
    created_at timestamp default current_timestamp
);
