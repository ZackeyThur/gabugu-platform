<template>
  <LoginLayout>
    <slot />
    <template #footer>
      <slot name="footer" />
    </template>
  </LoginLayout>
</template>

<script setup lang="ts">
import LoginLayout from './LoginLayout.vue'
</script>
