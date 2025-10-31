-- backend/schema.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(500),
    bio TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Categories table
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    image_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Recipes table
CREATE TABLE recipes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    featured_image_url VARCHAR(500),
    preparation_time INTEGER NOT NULL,
    cooking_time INTEGER NOT NULL,
    servings INTEGER NOT NULL,
    difficulty_level VARCHAR(20) CHECK (difficulty_level IN ('easy', 'medium', 'hard')),
    category_id UUID REFERENCES categories(id),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    price DECIMAL(10,2) DEFAULT 0,
    average_rating DECIMAL(3,2) DEFAULT 0,
    total_ratings INTEGER DEFAULT 0,
    like_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    is_published BOOLEAN DEFAULT FALSE
);

-- Ingredients table
CREATE TABLE ingredients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    quantity VARCHAR(100),
    unit VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Steps table
CREATE TABLE steps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    instruction TEXT NOT NULL,
    image_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Recipe images table
CREATE TABLE recipe_images (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    image_url VARCHAR(500) NOT NULL,
    is_featured BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Likes table
CREATE TABLE likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, recipe_id)
);

-- Bookmarks table
CREATE TABLE bookmarks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, recipe_id)
);

-- Comments table
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Ratings table
CREATE TABLE ratings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, recipe_id)
);

-- Purchases table
CREATE TABLE purchases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    amount DECIMAL(10,2) NOT NULL,
    chapa_transaction_id VARCHAR(255),
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Functions and Triggers

-- Function to update recipe average rating
CREATE OR REPLACE FUNCTION update_recipe_rating()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE recipes 
    SET 
        average_rating = (
            SELECT AVG(rating)::DECIMAL(3,2) 
            FROM ratings 
            WHERE recipe_id = NEW.recipe_id
        ),
        total_ratings = (
            SELECT COUNT(*) 
            FROM ratings 
            WHERE recipe_id = NEW.recipe_id
        )
    WHERE id = NEW.recipe_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for ratings
CREATE TRIGGER trigger_update_recipe_rating
    AFTER INSERT OR UPDATE OR DELETE ON ratings
    FOR EACH ROW
    EXECUTE FUNCTION update_recipe_rating();

-- Function to update like count
CREATE OR REPLACE FUNCTION update_recipe_like_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE recipes SET like_count = like_count + 1 WHERE id = NEW.recipe_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE recipes SET like_count = like_count - 1 WHERE id = OLD.recipe_id;
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Trigger for likes
CREATE TRIGGER trigger_update_like_count
    AFTER INSERT OR DELETE ON likes
    FOR EACH ROW
    EXECUTE FUNCTION update_recipe_like_count();

-- Function to ensure only one featured image per recipe
CREATE OR REPLACE FUNCTION ensure_single_featured_image()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.is_featured THEN
        UPDATE recipe_images 
        SET is_featured = FALSE 
        WHERE recipe_id = NEW.recipe_id AND id != NEW.id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for featured images
CREATE TRIGGER trigger_ensure_single_featured_image
    BEFORE INSERT OR UPDATE ON recipe_images
    FOR EACH ROW
    EXECUTE FUNCTION ensure_single_featured_image();

-- Search function
CREATE OR REPLACE FUNCTION search_recipes(
    search_query TEXT,
    category_filter UUID DEFAULT NULL,
    max_prep_time INTEGER DEFAULT NULL,
    ingredient_filter TEXT DEFAULT NULL,
    min_rating DECIMAL DEFAULT NULL
)
RETURNS TABLE(
    recipe_id UUID,
    title VARCHAR,
    description TEXT,
    featured_image_url VARCHAR,
    preparation_time INTEGER,
    cooking_time INTEGER,
    servings INTEGER,
    difficulty_level VARCHAR,
    average_rating DECIMAL,
    like_count INTEGER,
    user_id UUID,
    username VARCHAR,
    category_name VARCHAR,
    match_score NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        r.id,
        r.title,
        r.description,
        r.featured_image_url,
        r.preparation_time,
        r.cooking_time,
        r.servings,
        r.difficulty_level,
        r.average_rating,
        r.like_count,
        u.id as user_id,
        u.username,
        c.name as category_name,
        TS_RANK(
            TO_TSVECTOR('english', COALESCE(r.title, '') || ' ' || COALESCE(r.description, '')),
            PLAINTO_TSQUERY('english', search_query)
        ) as match_score
    FROM recipes r
    JOIN users u ON r.user_id = u.id
    JOIN categories c ON r.category_id = c.id
    WHERE 
        (search_query IS NULL OR search_query = '' OR 
         TO_TSVECTOR('english', COALESCE(r.title, '') || ' ' || COALESCE(r.description, '')) 
         @@ PLAINTO_TSQUERY('english', search_query))
        AND (category_filter IS NULL OR r.category_id = category_filter)
        AND (max_prep_time IS NULL OR (r.preparation_time + r.cooking_time) <= max_prep_time)
        AND (min_rating IS NULL OR r.average_rating >= min_rating)
        AND (ingredient_filter IS NULL OR EXISTS (
            SELECT 1 FROM ingredients i 
            WHERE i.recipe_id = r.id AND i.name ILIKE '%' || ingredient_filter || '%'
        ))
        AND r.is_published = TRUE
    ORDER BY match_score DESC, r.created_at DESC;
END;
$$ LANGUAGE plpgsql;