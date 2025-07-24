import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      // 修改为返回 Promise 的函数
      component: () => import("../views/Home.vue"),
    },
  ]
})

export default router