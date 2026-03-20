<script setup>
import { computed } from 'vue'

const appName = 'Linebackerr'
const tagline = computed(() => 'Hello World')
</script>

<template>
  <main class="app-shell d-flex align-items-center justify-content-center">
    <section class="brand-card text-center p-4 p-md-5 shadow-lg">
      <div class="icon-wrap mb-3">
        <FontAwesomeIcon icon="fa-solid fa-football" size="2x" />
      </div>
      <h1 class="display-5 fw-bold mb-2">{{ appName }}</h1>
      <p class="lead mb-0">{{ tagline }}</p>
    </section>
  </main>
</template>
