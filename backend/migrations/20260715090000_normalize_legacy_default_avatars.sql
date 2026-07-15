-- +goose Up
UPDATE `users`
SET `avatar` = ''
WHERE `avatar` = 'https://cdn.dianzaozao.com/avatars/f512f44051984823941dd0d214ed84f6.jpg';

-- +goose Down
-- Intentionally left empty: the migration cannot distinguish legacy defaults
-- from users who already had an empty avatar before the normalization.
