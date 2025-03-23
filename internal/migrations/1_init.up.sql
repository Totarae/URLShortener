CREATE TABLE urls (
                      id SERIAL PRIMARY KEY,
                      origin VARCHAR(100) NOT NULL UNIQUE,
                      shorten TEXT NOT NULL UNIQUE,
                      created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                      user_id TEXT
);