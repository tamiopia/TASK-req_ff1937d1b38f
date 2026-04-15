import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import HealthView from '@/views/HealthView.vue'
import LoginView from '@/views/LoginView.vue'

const routes = [
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/health',
    name: 'health',
    component: HealthView,
    meta: { public: true },
  },
  {
    path: '/login',
    name: 'login',
    component: LoginView,
    meta: { public: true },
  },
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('@/views/DashboardView.vue'),
    meta: { requiresAuth: true },
  },

  // ── Phase 3: User profile, preferences, address book ────────────────────────
  {
    path: '/profile',
    name: 'profile',
    component: () => import('@/views/ProfileView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/preferences',
    name: 'preferences',
    component: () => import('@/views/PreferencesView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/addresses',
    name: 'addresses',
    component: () => import('@/views/AddressBookView.vue'),
    meta: { requiresAuth: true },
  },

  // ── Phase 4: Service catalog & offerings ────────────────────────────────────
  {
    path: '/catalog',
    name: 'catalog',
    component: () => import('@/views/ServiceCatalogView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/catalog/new',
    name: 'catalog-new',
    component: () => import('@/views/ServiceOfferingFormView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/catalog/:id',
    name: 'catalog-detail',
    component: () => import('@/views/ServiceOfferingDetailView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/catalog/:id/edit',
    name: 'catalog-edit',
    component: () => import('@/views/ServiceOfferingFormView.vue'),
    meta: { requiresAuth: true },
  },

  // ── Phase 5: Tickets ────────────────────────────────────────────────────────
  {
    path: '/tickets',
    name: 'tickets',
    component: () => import('@/views/TicketListView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/tickets/new',
    name: 'ticket-new',
    component: () => import('@/views/TicketCreateView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/tickets/:id',
    name: 'ticket-detail',
    component: () => import('@/views/TicketDetailView.vue'),
    meta: { requiresAuth: true },
  },

  // ── Phase 6: Reviews & Q&A (embedded under catalog offerings) ───────────────
  {
    path: '/catalog/:id/reviews',
    name: 'offering-reviews',
    component: () => import('@/views/ReviewListView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/catalog/:id/qa',
    name: 'offering-qa',
    component: () => import('@/views/QAThreadView.vue'),
    meta: { requiresAuth: true },
  },

  // ── Phase 7: Notifications ──────────────────────────────────────────────────
  {
    path: '/notifications/outbox',
    name: 'notification-outbox',
    component: () => import('@/views/NotificationOutboxView.vue'),
    meta: { requiresAuth: true },
  },

  // ── Phase 8: Moderation ─────────────────────────────────────────────────────
  {
    path: '/moderation/queue',
    name: 'moderation-queue',
    component: () => import('@/views/ModerationQueueView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/moderation/violations',
    name: 'moderation-violations',
    component: () => import('@/views/ViolationHistoryView.vue'),
    meta: { requiresAuth: true },
  },

  // ── Phase 9: Privacy Center ─────────────────────────────────────────────────
  {
    path: '/privacy',
    name: 'privacy',
    component: () => import('@/views/PrivacyCenterView.vue'),
    meta: { requiresAuth: true },
  },

  // ── Phase 10: Data Operations (admin-only legal holds dashboard) ────────────
  {
    path: '/admin/legal-holds',
    name: 'legal-holds',
    component: () => import('@/views/LegalHoldView.vue'),
    meta: { requiresAuth: true },
  },

  // ── Security: HMAC key lifecycle (admin-only) ───────────────────────────────
  {
    path: '/admin/hmac-keys',
    name: 'hmac-keys',
    component: () => import('@/views/HMACKeysView.vue'),
    meta: { requiresAuth: true },
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

// Auth guard
router.beforeEach(async (to) => {
  const auth = useAuthStore()

  // Hydrate session on first navigation
  if (!auth.initialized) {
    await auth.fetchMe()
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }
})

export default router
