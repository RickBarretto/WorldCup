# Problem 2 - Multiplayer Distributed Card Game

## Context

A prototype of a **multiplayer card game** was initially developed using a **centralized architecture**, see [`v1.0.0`](https://github.com/RickBarretto/Eleventh/tree/v1.0.0). Where a single server managed game logic, player states, and communication. The prototype achieved early success, demonstrating the demand for online interactive experiences with social connectivity. 

However, as the player base expanded, the centralized model revealed critical **scalability limitations** and a **single point of failure**. To support broader adoption and ensure resilience, the system requires a **reengineering effort**. Migrating from a centralized model to a **distributed architecture**.

## Problem Statement

The task is to redesign the game infrastructure so that **multiple servers collaborate** in hosting matches and managing shared resources. The new architecture must achieve:

1. **Scalability** – Distribute processing and management loads across servers to support a significantly larger number of players.
2. **Fault Tolerance** – Eliminate the central server as a single point of failure, maintaining continuous availability even under server crashes.
3. **Consistency and Fairness** – Ensure the integrity of the game state and equitable card distribution across a distributed environment.
4. **New Functionality** – Introduce a **card trading feature** between players, extending gameplay dynamics.

This transformation must guarantee robust, fair, and uninterrupted gameplay, despite the complexities of concurrency and distributed coordination.

## Constraints

The implementation is subject to the following requirements:

1. **Containerization**
   * All components, including servers and clients, must run in **Docker containers**, supporting multiple concurrent instances for testing.
2. **Decentralized Architecture**
   * The system must comprise **multiple collaborating servers**, each potentially responsible for a region or group of players.
   * Servers must be deployed in **separate containers and distinct laboratory machines**, emulating real distributed environments.
3. **Server-to-Server Communication**
   * Implemented through a **REST API** designed by the team.
   * Testable using tools such as **Insomnia** or **Postman**.
4. **Server-to-Client Communication**
   * Based on the **Publisher-Subscriber model**, enabling real-time interaction.
   * Third-party libraries may be used.
5. **Distributed Card Management**
   * Card packs serve as a **global shared stock**.
   * The system must ensure:
     * **Fair allocation** – only one player receives a given pack.
     * **Consistency** – no duplication or loss of cards.
     * **Resilience** – no reliance on a single server for card management.
6. **Cross-Server Matches**
   * Players connected to different servers must be able to duel in 1v1 matches.
   * The uniqueness of match pairings must be preserved.
7. **Fault Tolerance**
   * The system must remain operational despite one or more server failures.
   * Failures should minimally affect gameplay and maintain data consistency.
8. **Testing**
   * The solution must include **software tests** addressing distributed concurrency (e.g., simultaneous access to card packs) and fault scenarios (e.g., server crashes).

## Rules

* Projects are to be developed in groups of up to **two students**.
* The **final deadline** is **October 23, 2025**.
* Deliverables include:
  * **Source code on GitHub**, fully commented, with a README and test scripts.
  * A **written report** (maximum 8 pages, SBC format), documenting concepts, design choices, and justifications.
* Presentations will take place in the **Laboratory of Networks and Distributed Systems (LARSID)**. Each group will have **25 minutes** to demonstrate the system and answer technical questions.
* Late submissions will be penalized, and plagiarism or identical projects will result in a grade of zero.

## Evaluation Criteria

The final grade will consist of:

1. **Tutorial performance** – 30%
2. **Written report** – 20%
3. **Final product on GitHub** – 50%
