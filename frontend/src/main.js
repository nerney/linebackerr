import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createBootstrap } from 'bootstrap-vue-next'
import { library } from '@fortawesome/fontawesome-svg-core'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faFootball } from '@fortawesome/free-solid-svg-icons'
import { faGithub } from '@fortawesome/free-brands-svg-icons'

import App from './App.vue'
import router from './router'

import 'bootstrap/dist/css/bootstrap.min.css'
import 'bootstrap'
import 'bootstrap-vue-next/dist/bootstrap-vue-next.css'
import './app.css'

library.add(faFootball, faGithub)

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(createBootstrap())
app.component('FontAwesomeIcon', FontAwesomeIcon)

app.mount('#app')
