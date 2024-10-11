CREATE TABLE rent_requests (
    id SERIAL PRIMARY KEY,
    renter_id INTEGER NOT NULL,
    owner_id INTEGER NOT NULL,
    post_id INTEGER NOT NULL,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    total_price INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    payment_status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES "user-service".users(id) ON DELETE CASCADE,
    FOREIGN KEY (post_id) REFERENCES "post-service".posts(id) ON DELETE CASCADE,
    FOREIGN KEY (renter_id) REFERENCES "user-service".users(id) ON DELETE CASCADE
);
