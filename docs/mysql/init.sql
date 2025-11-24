-- create the databases
CREATE
DATABASE IF NOT EXISTS `dlock`;

create table dlock.distributed_locks
(
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    `key`      VARCHAR(256) NOT NULL,
    value      CHAR(64)     NOT NULL,
    status     TINYINT      NOT NULL,
    version    BIGINT       NOT NULL,
    expiration BIGINT       NOT NULL,
    utime      BIGINT       NOT NULL,
    ctime      BIGINT       NOT NULL,
    INDEX      idx_utime(`utime`),
    UNIQUE idx_key(`key`)
);


