{
  "_note": "The content system for the cookbook. Three worlds, so each recipe looks like it came from a real product rather than a fixture. All brands, people and copy are invented; nothing is taken from a real company.",

  "worlds": {
    "imagor": {
      "who": "The project itself. Developer-facing cards.",
      "recipes": ["announcement", "tip", "quote", "blog-og"],
      "brand": "imagor",
      "voice": "Precise, no marketing adjectives. States a property, not a benefit.",
      "mark": "imagor"
    },
    "maison": {
      "who": "An invented leather-goods shop. Retail cards.",
      "recipes": ["product", "product-og", "product-grid", "news-announcement"],
      "brand": "MAISON",
      "voice": "Quiet and specific. Names the material and the construction, never 'premium' or 'luxury'.",
      "mark": "MAISON"
    },
    "harbour": {
      "who": "An invented infrastructure company. Cards that need a person.",
      "recipes": ["avatar-card", "testimonial", "quote-card", "event"],
      "brand": "Harbour",
      "voice": "Plain engineering speech. What changed, and by how much.",
      "mark": "Harbour"
    }
  },

  "people": {
    "avatar-card": { "name": "Mei Lam", "role": "Platform engineering", "handle": "@meilam" },
    "quote-card": { "who": "Mei Lam", "role": "Platform engineering, Harbour" },
    "testimonial": { "name": "Mei Lam", "company": "Platform engineering, Harbour" },
    "event": { "venue": "Harbour office, 6pm" }
  },

  "rules": [
    "No real company or product names anywhere.",
    "No marketing adjectives: premium, luxury, seamless, powerful, robust.",
    "Prices are plausible for the thing described, in HKD.",
    "A card that needs a date gets one; no empty values.",
    "Product names describe construction, as a real catalogue would."
  ]
}
