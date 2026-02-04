// Klaro Cookie Consent Configuration
// Documentation: https://klaro.org/docs/

window.klaroConfig = {
    // Version for consent storage (bump when config changes significantly)
    version: 1,

    // Element to mount Klaro to (defaults to body)
    elementID: 'klaro',

    // Storage method for consent
    storageMethod: 'cookie',
    storageName: 'klaro',

    // Cookie domain (leave empty for current domain)
    cookieDomain: '',

    // Cookie expiration in days
    cookieExpiresAfterDays: 365,

    // Privacy policy URL
    privacyPolicy: '/privacy-policy',

    // Default consent state (false = opt-in required)
    default: true,

    // Must consent before using the site
    mustConsent: true,

    // Accept all by default when clicking accept
    acceptAll: true,

    // Hide decline all button
    hideDeclineAll: false,

    // Hide learn more link
    hideLearnMore: false,

    // Disable cookie banner popup (set to false to show it)
    disablePoweredBy: true,

    // Translations
    translations: {
        en: {
            consentModal: {
                title: 'Help us get better!',
                description: "We're a community-run makerspace, and we'd love your help! By enabling cookies, you help us learn what interests our visitors and how we can improve. Your consent directly benefits the whole community.",
            },
            consentNotice: {
                title: 'Help us get better!',
                description: "We're a community-run makerspace. Enabling cookies helps us understand what you're interested in so we can improve for everyone's benefit. Your consent helps the whole community!",
                learnMore: 'Learn more',
                changeDescription: 'There were changes since your last visit, please update your consent.',
            },
            ok: 'Happy to help!',
            save: 'Save',
            decline: 'No thanks',
            close: 'Close',
            acceptAll: 'Happy to help!',
            acceptSelected: 'Accept selected',
            purposes: {
                analytics: {
                    title: 'Analytics',
                    description: 'Help us understand how visitors use our site so we can make it better for the community.',
                },
            },
            googleAnalytics: {
                title: 'Google Analytics',
                description: 'Helps us see which pages and content resonate with our community so we can focus on what matters most.',
            },
        },
    },

    // Services/Apps that require consent
    services: [
        {
            name: 'googleAnalytics',
            title: 'Google Analytics',
            purposes: ['analytics'],
            cookies: [
                /^_ga/,
                /^_gid/,
            ],
            required: false,
            optOut: false,
            default: true,
            onlyOnce: false,
        },
    ],
};