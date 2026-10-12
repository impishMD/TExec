import Vue from 'vue';
import { vuetifyThemes } from '@/lib/appearance';
import Vuetify from 'vuetify/lib';
import OpenTofuIcon from '@/components/OpenTofuIcon.vue';
import PulumiIcon from '@/components/PulumiIcon.vue';
import TerragruntIcon from '@/components/TerragruntIcon.vue';
import HashicorpVaultIcon from '@/components/HashicorpVaultIcon.vue';
import TelegramIcon from '@/components/TelegramIcon.vue';
import {
  cs, de, en, es, fr, it, ja, ko, nl, pl, pt, ru, zhHans, zhHant,
} from 'vuetify/lib/locale';
import DvlsIcon from '../components/DvlsIcon.vue';
import AwsSmIcon from '../components/AwsSmIcon.vue';
import AzureKvIcon from '../components/AzureKvIcon.vue';
import i18n from './i18';

Vue.use(Vuetify);

export default new Vuetify({
  lang: {
    current: i18n.locale,
    locales: {
      cs, de, en, es, fr, it, ja, ko, nl, pl, pt, pt_br: pt, ru, zh_cn: zhHans, zh_tw: zhHant,
    },
  },
  theme: {
    dark: localStorage.getItem('darkMode') === '1',
    themes: vuetifyThemes('texec'),
  },
  icons: {
    values: {
      telegram: {
        component: TelegramIcon,
      },
      tofu: {
        component: OpenTofuIcon,
      },
      pulumi: {
        component: PulumiIcon,
      },
      terragrunt: {
        component: TerragruntIcon,
      },
      hashicorp_vault: {
        component: HashicorpVaultIcon,
      },
      dvls: {
        component: DvlsIcon,
      },
      aws_sm: {
        component: AwsSmIcon,
      },
      azure_kv: {
        component: AzureKvIcon,
      },
    },
  },
});
