# Database Migration: Search Keys

This document describes how to run the search keys migration to add fuzzy search capabilities to existing player and club data.

## What This Migration Does

The search keys migration adds precomputed search fields to existing players and clubs in MongoDB:

- **normalized**: Lowercase text with Swedish diacritics removed (å→a, ä→a, ö→o)
- **prefixes**: Character prefixes for autocomplete matching
- **trigrams**: Character trigrams for typo tolerance

## Running the Migration

### Option 1: Using the Migration Command (Recommended)

```bash
cd backend
go run cmd/migrate/main.go add-search-keys
```

### Option 2: Manual MongoDB Update

If the Go command doesn't work, you can run this MongoDB script directly:

```javascript
// Connect to your MongoDB database
use pingis

// Add search keys to players
db.players.find({search_keys: {$exists: false}}).forEach(function(player) {
    if (player.display_name) {
        var normalized = player.display_name.toLowerCase()
            .replace(/å/g, 'a')
            .replace(/ä/g, 'a') 
            .replace(/ö/g, 'o')
            .replace(/é/g, 'e');
        
        var words = normalized.split(/\s+/);
        var prefixes = [];
        var trigrams = [];
        
        // Generate prefixes
        words.forEach(function(word) {
            for (var i = 2; i <= Math.min(word.length, 6); i++) {
                prefixes.push(word.substring(0, i));
            }
        });
        
        // Generate trigrams
        words.forEach(function(word) {
            var padded = '  ' + word + '  ';
            for (var i = 0; i <= padded.length - 3; i++) {
                trigrams.push(padded.substring(i, i + 3));
            }
        });
        
        db.players.updateOne(
            {_id: player._id},
            {$set: {
                search_keys: {
                    normalized: normalized,
                    prefixes: prefixes,
                    trigrams: trigrams
                }
            }}
        );
        
        print('Updated player: ' + player.display_name);
    }
});

// Add search keys to clubs
db.clubs.find({search_keys: {$exists: false}}).forEach(function(club) {
    if (club.name) {
        var normalized = club.name.toLowerCase()
            .replace(/å/g, 'a')
            .replace(/ä/g, 'a')
            .replace(/ö/g, 'o')
            .replace(/é/g, 'e');
        
        var words = normalized.split(/\s+/);
        var prefixes = [];
        var trigrams = [];
        
        // Generate prefixes
        words.forEach(function(word) {
            for (var i = 2; i <= Math.min(word.length, 6); i++) {
                prefixes.push(word.substring(0, i));
            }
        });
        
        // Generate trigrams
        words.forEach(function(word) {
            var padded = '  ' + word + '  ';
            for (var i = 0; i <= padded.length - 3; i++) {
                trigrams.push(padded.substring(i, i + 3));
            }
        });
        
        db.clubs.updateOne(
            {_id: club._id},
            {$set: {
                search_keys: {
                    normalized: normalized,
                    prefixes: prefixes,
                    trigrams: trigrams
                }
            }}
        );
        
        print('Updated club: ' + club.name);
    }
});

// Create indexes for better search performance
db.players.createIndex({"search_keys.normalized": 1});
db.players.createIndex({"search_keys.prefixes": 1});
db.players.createIndex({"search_keys.trigrams": 1});

db.clubs.createIndex({"search_keys.normalized": 1});
db.clubs.createIndex({"search_keys.prefixes": 1});
db.clubs.createIndex({"search_keys.trigrams": 1});

print('Migration completed successfully!');
```

## Verifying the Migration

After running the migration, you can verify it worked by checking a few documents:

```javascript
// Check that players have search_keys
db.players.findOne({search_keys: {$exists: true}})

// Check that clubs have search_keys  
db.clubs.findOne({search_keys: {$exists: true}})

// Check indexes were created
db.players.getIndexes()
db.clubs.getIndexes()
```

## Migration Status

The migration is **idempotent** - it's safe to run multiple times. It will only update documents that don't already have search_keys.

## Troubleshooting

If you encounter issues:

1. **MongoDB Connection**: Ensure MongoDB is running and accessible
2. **Database Name**: Make sure you're connected to the correct database (usually `pingis`)
3. **Permissions**: Ensure your MongoDB user has write permissions
4. **Memory**: For large datasets, the migration may take several minutes

## Next Steps

After the migration completes:

1. Restart the Klubbspel backend application
2. Test the enhanced search functionality in the frontend
3. Verify that Swedish name variations work (e.g., "tomas" finds "Thomas")