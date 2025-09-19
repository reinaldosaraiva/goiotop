
### **Template de Documento de Requisitos de Software**

**Nome do Projeto:** [Inserir o nome do projeto]

**Versão do Documento:** 1.0

**Data da Última Atualização:** [DD/MM/AAAA]

**Autor:** [Nome do Analista de Requisitos]

---

### **1. Introdução e Visão Geral do Sistema**

* **1.1. Missão e Propósito do Sistema:**
    * Descrever o objetivo principal do software e o problema de negócio que ele se propõe a resolver. Explicar o papel estratégico do sistema dentro da organização ou para seus usuários finais.
* **1.2. Escopo do Projeto:**
    * Definir claramente o que o sistema fará e, igualmente importante, o que ele não fará. Listar os principais módulos e funcionalidades que estão dentro do escopo.
* **1.3. Contexto e Integração:**
    * Descrever o ecossistema em que o software operará, incluindo outros sistemas com os quais ele precisará interagir ou se integrar.
* **1.4. Glossário de Termos e Siglas:**
    * Listar e definir todos os termos técnicos, acrônimos e jargões específicos do domínio do negócio para garantir um entendimento comum entre todas as partes interessadas.

### **2. Personas de Usuário**

* **2.1. Descrição das Personas:**
    * Identificar e descrever detalhadamente cada perfil de usuário que irá interagir com o sistema. O objetivo é criar uma representação clara de quem são os usuários, suas responsabilidades e seus objetivos ao usar o software.

    * **Exemplo de Tabela de Personas:**
| Persona | Responsabilidade Principal | Interação Primária com o Sistema | Necessidades Chave de Informação |
| :--- | :--- | :--- | :--- |
| **[Nome da Persona 1]** | [Descrição das responsabilidades principais] | [Ações que a persona realiza no sistema] | [Informações cruciais que a persona precisa] |
| **[Nome da Persona 2]** | [Descrição das responsabilidades principais] | [Ações que a persona realiza no sistema] | [Informações cruciais que a persona precisa] |
| **[Nome da Persona 3]** | [Descrição das responsabilidades principais] | [Ações que a persona realiza no sistema] | [Informações cruciais que a persona precisa] |

### **3. Requisitos Funcionais (Épicos e Histórias de Usuário)**

* **3.1. Visão Geral dos Épicos:**
    * Agrupar as funcionalidades do sistema em grandes blocos chamados "Épicos". Cada épico representa uma área de funcionalidade principal do sistema.
* **3.2. Detalhamento por Épico:**
    * Para cada épico, detalhar as "Histórias de Usuário" específicas, seguindo o formato padrão.

    * **Épico 1: [Nome do Épico, ex: Gestão de Ocorrências]**
        * **Histórias de Usuário para a [Persona 1]:**
            * **US-ID-001:** Como um **[Persona]**, eu quero **[realizar uma ação]** para que **[eu obtenha um benefício]**.
            * **US-ID-002:** Como um **[Persona]**, eu quero **[realizar uma ação]** para que **[eu obtenha um benefício]**.
        * **Histórias de Usuário para a [Persona 2]:**
            * **US-ID-003:** Como um **[Persona]**, eu quero **[realizar uma ação]** para que **[eu obtenha um benefício]**.
            * **US-ID-004:** Como um **[Persona]**, eu quero **[realizar uma ação]** para que **[eu obtenha um benefício]**.

    * **Épico 2: [Nome do Épico, ex: Gestão de Recursos e Frota]**
        * **(Continuar com as histórias de usuário para cada persona relevante)**

### **4. Requisitos Não Funcionais**

* Definir os atributos de qualidade e as restrições do sistema. Estes requisitos descrevem "como" o sistema deve funcionar.

* **4.1. Confiabilidade e Disponibilidade:**
    * **Requisito:** O sistema deve ter uma disponibilidade de XX.XX%.
    * **Justificativa:** A criticidade da operação exige tempo de inatividade mínimo.
* **4.2. Desempenho:**
    * **Requisito:** As ações críticas do usuário (ex: salvar um registro) devem ter um tempo de resposta inferior a X segundos.
    * **Justificativa:** A agilidade é crucial para a eficiência operacional.
* **4.3. Segurança:**
    * **Requisito:** O acesso ao sistema deve ser controlado por papéis e permissões (RBAC).
    * **Requisito:** Todos os dados sensíveis devem ser criptografados em trânsito e em repouso.
    * **Requisito:** Todas as ações significativas devem ser registradas em um log de auditoria imutável.
* **4.4. Escalabilidade:**
    * **Requisito:** A arquitetura deve suportar um crescimento de X% no número de usuários e volume de dados nos próximos Y anos.
* **4.5. Usabilidade:**
    * **Requisito:** A interface deve ser intuitiva, minimizando a carga cognitiva e o número de cliques para tarefas comuns.
* **4.6. Conformidade Legal:**
    * **Requisito:** O sistema deve estar em conformidade com [mencionar leis e regulamentações aplicáveis, ex: LGPD].

### **5. Modelo de Dados**

* **5.1. Diagrama Entidade-Relacionamento (Opcional):**
    * Incluir um diagrama visual que represente as principais entidades de dados e seus relacionamentos.
* **5.2. Descrição das Entidades de Dados:**
    * Descrever as principais tabelas ou entidades do banco de dados, seus atributos chave e como se relacionam.

    * **Exemplo de Tabela de Entidades:**
| Entidade | Atributos Chave | Relacionamentos |
| :--- | :--- | :--- |
| **[Entidade 1]** | ID (PK), Atributo1, Atributo2, ID\_Entidade2 (FK) | Relaciona-se com [Entidade 2], [Entidade 3]. |
| **[Entidade 2]** | ID (PK), AtributoA, AtributoB | Uma [Entidade 2] pode ter muitas [Entidade 1]. |

### **6. Apêndices**

* Incluir quaisquer informações adicionais relevantes, como protótipos de tela (wireframes), fluxogramas de processos de negócio, ou referências a documentos externos.

* **Exemplo de Apêndice: Fluxograma do Processo de Atendimento de Ocorrência**

    Este fluxograma ilustra as etapas do processo de atendimento de uma ocorrência, desde o recebimento da chamada até o fechamento do caso. Ele ajuda a visualizar a sequência de ações e as decisões que precisam ser tomadas pelos diferentes usuários do sistema.
    

    ```mermaid
    flowchart TD
        A[Início: Chamada de emergência recebida] --> B[Triagem Inicial TARM]
        B --> C{Necessita de Regulação Médica?}
        C -- Sim --> D[Regulação Médica Médico Regulador]
        C -- Não --> N[TARM fornece orientações e encerra]
        D --> E{Decisão: Tipo de Recurso?}
        E -- USB --> F[Despacho do Recurso Rádio Operador]
        E -- USA --> F
        E -- Outro --> F
        F --> G[Equipe em Deslocamento]
        G --> H[Chegada ao Local]
        H --> I[Atendimento e Transporte]
        I --> J[Chegada ao Destino]
        J --> K[Fim do Atendimento no Sistema]
        K --> L[Fechamento do Caso]
        
        N --> M[Fim]
        L --> M;
    ```
