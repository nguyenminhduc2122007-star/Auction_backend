-- Auction System Database Schema
-- Auto-increment integer IDs (1, 2, 3, ...)

-- =====================================================
-- USERS TABLE
-- =====================================================
CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_type ENUM('Bidder', 'Seller', 'Admin') NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    address TEXT,
    city VARCHAR(100),
    state VARCHAR(100),
    postal_code VARCHAR(20),
    country VARCHAR(100),
    profile_picture_url VARCHAR(500),
    is_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    rating DECIMAL(3, 2) DEFAULT 0,
    total_auctions_won INT DEFAULT 0,
    total_auctions_created INT DEFAULT 0,
    total_items_sold INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,

    INDEX idx_email (email),
    INDEX idx_user_type (user_type),
    INDEX idx_is_active (is_active),
    INDEX idx_created_at (created_at),
    INDEX idx_rating (rating)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================================
-- ITEMS TABLE
-- =====================================================
CREATE TABLE items (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    item_type ENUM('Electronics', 'Art', 'Vehicle') NOT NULL,
    seller_id BIGINT UNSIGNED NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status ENUM('Draft', 'Pending', 'Approved', 'Rejected', 'Listed', 'Sold', 'Archived') DEFAULT 'Draft',
    category VARCHAR(100),
    base_price DECIMAL(15, 2) NOT NULL,
    `condition` ENUM('New', 'Like New', 'Good', 'Fair', 'Poor') DEFAULT 'Good',
    image_url VARCHAR(500),
    brand VARCHAR(100),
    model VARCHAR(100),
    specifications VARCHAR(500),
    warranty_months INT,
    artist_name VARCHAR(255),
    year_created INT,
    medium VARCHAR(100),
    dimensions VARCHAR(100),
    certificate_of_authenticity BOOLEAN DEFAULT FALSE,
    make VARCHAR(100),
    model_year INT,
    mileage INT,
    vin VARCHAR(100),
    fuel_type VARCHAR(50),
    transmission VARCHAR(50),
    color VARCHAR(100),
    rejection_reason TEXT,
    admin_notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,

    FOREIGN KEY (seller_id) REFERENCES users(id),
    INDEX idx_seller_id (seller_id),
    INDEX idx_item_type (item_type),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_base_price (base_price)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================================
-- AUCTIONS TABLE
-- =====================================================
CREATE TABLE auctions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    item_id BIGINT UNSIGNED NOT NULL,
    seller_id BIGINT UNSIGNED NOT NULL,
    starting_price DECIMAL(15, 2) NOT NULL,
    reserve_price DECIMAL(15, 2),
    current_highest_bid DECIMAL(15, 2) DEFAULT 0,
    status ENUM('UPCOMING', 'ACTIVE', 'ENDED', 'CANCELLED') DEFAULT 'UPCOMING',
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    winner_id BIGINT UNSIGNED,
    final_price DECIMAL(15, 2),
    total_bids INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,

    FOREIGN KEY (item_id) REFERENCES items(id),
    FOREIGN KEY (seller_id) REFERENCES users(id),
    FOREIGN KEY (winner_id) REFERENCES users(id),
    INDEX idx_item_id (item_id),
    INDEX idx_seller_id (seller_id),
    INDEX idx_winner_id (winner_id),
    INDEX idx_status (status),
    INDEX idx_start_time (start_time),
    INDEX idx_end_time (end_time),
    UNIQUE INDEX idx_item_auction (item_id),
    INDEX idx_price_range (starting_price, current_highest_bid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================================
-- BID_TRANSACTIONS TABLE
-- =====================================================
CREATE TABLE bid_transactions (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    auction_id BIGINT UNSIGNED NOT NULL,
    bidder_id BIGINT UNSIGNED NOT NULL,
    bid_amount DECIMAL(15, 2) NOT NULL,
    status ENUM('Active', 'Outbid', 'Won', 'Cancelled') DEFAULT 'Active',
    bid_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (auction_id) REFERENCES auctions(id),
    FOREIGN KEY (bidder_id) REFERENCES users(id),
    INDEX idx_auction_id (auction_id),
    INDEX idx_bidder_id (bidder_id),
    INDEX idx_status (status),
    INDEX idx_bid_time (bid_time),
    INDEX idx_auction_bidder (auction_id, bidder_id),
    INDEX idx_amount (bid_amount)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================================
-- AUDIT LOG TABLE
-- =====================================================
CREATE TABLE audit_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id BIGINT UNSIGNED,
    changes JSON,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id),
    INDEX idx_user_id (user_id),
    INDEX idx_entity_type (entity_type),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- =====================================================
-- Initial Admin User (password: admin123)
-- =====================================================
INSERT INTO users (
    id, user_type, email, full_name, password_hash, is_verified, is_active
) VALUES (
    1,
    'Admin',
    'admin@auction.com',
    'System Admin',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcg7b3XeKeUxWdeS86E36gZvWFm',
    TRUE,
    TRUE
);
